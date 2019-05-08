package promquery

import (
	"context"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/solo-io/go-utils/errors"
	"sync"
	"time"
)

// stream of values for a polled query
type Results <-chan float64
type ResultsPush chan float64

type pubSub struct {
	closeChannels   map[Results]ResultsPush
	subscriberCache []ResultsPush
	subscriberLock  sync.RWMutex
}

func newPubsub() *pubSub {
	return &pubSub{closeChannels: make(map[Results]ResultsPush)}
}

func (r *pubSub) close() {
	r.subscriberLock.Lock()
	defer r.subscriberLock.Unlock()
	for _, subscriber := range r.subscriberCache {
		close(subscriber)
	}
	r.subscriberCache = nil
	r.closeChannels = make(map[Results]ResultsPush)
}

func (r *pubSub) subscribe() Results {
	r.subscriberLock.Lock()
	defer r.subscriberLock.Unlock()
	c := make(chan float64, 10)
	r.subscriberCache = append(r.subscriberCache, c)
	return c
}

func (r *pubSub) unsubscribe(c Results) {
	r.subscriberLock.Lock()
	defer r.subscriberLock.Unlock()
	closeChan, ok := r.closeChannels[c]
	if !ok {
		return
	}
	for i, subscriber := range r.subscriberCache {
		if subscriber == closeChan {
			delete(r.closeChannels, c)
			r.subscriberCache = append(r.subscriberCache[:i], r.subscriberCache[i+1:]...)
			return
		}
	}
}

func (r *pubSub) publish(ctx context.Context, result float64) {
	r.subscriberLock.RLock()
	defer r.subscriberLock.RUnlock()
	for _, subscriber := range r.subscriberCache {
		select {
		case <-ctx.Done():
			return
		case subscriber <- result:
		}
	}
}

// Publish results of Prometheus Queries on an interval, notifying subscribers of each query result
type QueryPubSub struct {
	client promv1.API

	// maintain a pubsub for each query
	queryPubSubs map[string]*pubSub
	access       sync.RWMutex

	// poll each query on this interval
	pollingInterval time.Duration
}

var defaultPollingInterval = time.Second * 5

func NewQueryPubSub(promClient promv1.API, customPollingInterval time.Duration) *QueryPubSub {
	interval := defaultPollingInterval
	if customPollingInterval != 0 {
		interval = customPollingInterval
	}
	return &QueryPubSub{
		client:          promClient,
		pollingInterval: interval,
		queryPubSubs:    make(map[string]*pubSub),
	}
}

func (c *QueryPubSub) queryScalar(ctx context.Context, query string) (float64, error) {
	result, err := c.client.Query(ctx, query, time.Now())
	if err != nil {
		return 0, err
	}
	scalar, ok := result.(*model.Scalar)
	if !ok {
		return 0, errors.Errorf("query result was %s, only scalar values supported", result.Type())
	}
	return float64(scalar.Value), nil
}

// will poll the query for the given interval until the ctx is cancelled
// subscribers who are watching this query will be notified on every tick
func (c *QueryPubSub) BeginPolling(ctx context.Context, query string) <-chan error {
	errs := make(chan error)

	go func() {
		defer close(errs)
		// remove all subscribers for this query when this function exits
		defer func() {
			c.access.Lock()
			if queryPubSub, ok := c.queryPubSubs[query]; ok {
				queryPubSub.close()
			}
			delete(c.queryPubSubs, query)
			c.access.Unlock()
		}()

		tick := time.NewTicker(c.pollingInterval)
		for {

			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				val, err := c.queryScalar(ctx, query)
				if err != nil {
					select {
					case <-ctx.Done():
						return
					case errs <- err:
					}
					continue
				}
				c.notifySubscribers(ctx, query, val)
			}
		}
	}()
	return errs
}

func (c *QueryPubSub) notifySubscribers(ctx context.Context, query string, val float64) {
	queryPubSub, ok := c.getPubSub(query)
	if ok {
		queryPubSub.publish(ctx, val)
	}
}

func (c *QueryPubSub) getPubSub(query string) (*pubSub, bool) {
	c.access.RLock()
	queryPubSub, ok := c.queryPubSubs[query]
	c.access.RUnlock()
	return queryPubSub, ok
}

func (c *QueryPubSub) Subscribe(query string) Results {
	queryPubSub, ok := c.getPubSub(query)
	if !ok {
		queryPubSub = newPubsub()
		c.access.Lock()
		c.queryPubSubs[query] = queryPubSub
		c.access.Unlock()
	}
	return queryPubSub.subscribe()
}

func (c *QueryPubSub) Unsubscribe(query string, results Results) {
	queryPubSub, ok := c.getPubSub(query)
	if !ok {
		return
	}
	queryPubSub.unsubscribe(results)
}
