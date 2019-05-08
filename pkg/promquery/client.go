package promquery

import (
	"context"
	"github.com/cskr/pubsub"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"time"
)

type QueryResult struct {
	Value model.Value
}

type ResultStream <-chan QueryResult

func ToResultStream(ctx context.Context, untyped <-chan interface{}) ResultStream {
	qs := make(chan QueryResult)
	go func() {
		defer close(qs)
		for {
			select {
			case <-ctx.Done():
				return
			case val, ok := <-untyped:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case qs <- val.(QueryResult):
				}
			}
		}
	}()
	return qs
}

// caches ongoing prometheus query results and publishes them to a list of subscribers
type PromCache interface {
	// publish the results of a query to a topic (query = topic)
	// when ctx is canceled, close this stream
	PublishQuery(ctx context.Context, query string, interval, duration time.Duration) error

	// subscribe to query stream
	Subscribe(ctx context.Context, query string) (ResultStream, <-chan error, error)

	// unsubscribe from query stream
	Unsubscribe(ctx context.Context, query string, results ResultStream) error
}

type promQueryCache struct {
	client promv1.API
	ps *pubsub.PubSub
}

func (p *promQueryCache) WatchQuery(ctx context.Context, query string, startTime time.Time, interval, duration time.Duration) (<-chan promv1.RuleType, <-chan error, error) {
	v, err := p.client.Query(ctx, query, startTime)
	if err != nil {
		return nil, err
	}

}

func NewPromClient(url string) (PromCache, error) {
	baseClient, err := promapi.NewClient(promapi.Config{Address: url})
	if err != nil {
		return nil, err
	}
	return &promQueryCache{client: promv1.NewAPI(baseClient), ps: pubsub.New(0)}, nil
}
