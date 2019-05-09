package promquery

import (
	"context"
	"sync"
	"time"

	"github.com/solo-io/go-utils/contextutils"

	"github.com/prometheus/common/model"
	"github.com/solo-io/go-utils/errors"
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

func (r *pubSub) active() bool {
	r.subscriberLock.RLock()
	defer r.subscriberLock.RUnlock()
	return len(r.subscriberCache) > 0
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

type QueryPubSub interface {
	Subscribe(query string) Results
	Unsubscribe(query string, results Results)
}

// Publish results of Prometheus Queries on an interval, notifying subscribers of each query result
type queryPubSub struct {
	rootCtx context.Context

	client QueryClient

	// maintain a pubsub for each query
	queryPubSubs map[string]*pubSub
	access       sync.RWMutex

	// poll each query on this interval
	pollingInterval time.Duration
}

var defaultPollingInterval = time.Second * 5

// the only method we need from the prometheus client
type QueryClient interface {
	Query(ctx context.Context, query string, ts time.Time) (model.Value, error)
}

func NewQueryPubSub(rootCtx context.Context, promClient QueryClient, customPollingInterval time.Duration) QueryPubSub {
	ctx := contextutils.WithLogger(rootCtx, "prometheus-query-pubsub")
	interval := defaultPollingInterval
	if customPollingInterval != 0 {
		interval = customPollingInterval
	}
	return &queryPubSub{
		client:          promClient,
		pollingInterval: interval,
		queryPubSubs:    make(map[string]*pubSub),
		rootCtx:         ctx,
	}
}

func (c *queryPubSub) queryScalar(ctx context.Context, query string) (float64, error) {
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
// watch errors are currently logged rather than returned on a channel
func (c *queryPubSub) beginPolling(query string) {
	go func() {
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
			case <-c.rootCtx.Done():
				return
			case <-tick.C:
				c.access.RLock()
				_, queryStillAcitve := c.queryPubSubs[query]
				c.access.RUnlock()
				if !queryStillAcitve {
					return
				}
				val, err := c.queryScalar(c.rootCtx, query)
				if err != nil {
					contextutils.LoggerFrom(c.rootCtx).Errorf("failed performing query on prometheus: %v", err)
					continue
				}
				c.notifySubscribers(c.rootCtx, query, val)
			}
		}
	}()
}

func (c *queryPubSub) notifySubscribers(ctx context.Context, query string, val float64) {
	queryPubSub, ok := c.getPubSub(query)
	if ok {
		queryPubSub.publish(ctx, val)
	}
}

func (c *queryPubSub) getPubSub(query string) (*pubSub, bool) {
	c.access.RLock()
	queryPubSub, ok := c.queryPubSubs[query]
	c.access.RUnlock()
	return queryPubSub, ok
}

func (c *queryPubSub) Subscribe(query string) Results {
	queryPubSub, ok := c.getPubSub(query)
	if !ok {
		c.beginPolling(query)
		queryPubSub = newPubsub()
		c.access.Lock()
		c.queryPubSubs[query] = queryPubSub
		c.access.Unlock()
	}
	return queryPubSub.subscribe()
}

func (c *queryPubSub) Unsubscribe(query string, results Results) {
	queryPubSub, ok := c.getPubSub(query)
	if !ok {
		return
	}
	queryPubSub.unsubscribe(results)

	// remove the query from the cache completely if all subscribers unsub
	if !queryPubSub.active() {
		queryPubSub.close()
		c.access.Lock()
		delete(c.queryPubSubs, query)
		c.access.Unlock()
	}
}
