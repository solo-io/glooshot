package promquery

import (
	"github.com/cskr/pubsub"
	"context"
	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	v1 "k8s.io/api/core/v1"
	"strings"
	"sync"
	"time"
)

type Metric struct {

}

type PromClient interface{
	WatchQuery(ctx context.Context, query string, interval, duration time.Duration) ( <-chan model.Value, <-chan error, error)

}

type promClient struct {
	client promv1.API

	ps *pubsub.PubSub
}

func (p *promClient) WatchQuery(ctx context.Context, query string, interval, duration time.Duration) (<-chan promv1.RuleType, <-chan error, error) {
	v, err := p.client.Query()

}

func NewPromClient(url string) (PromClient, error) {
	baseClient, err := promapi.NewClient(promapi.Config{Address: url})
	if err != nil {
		return nil, err
	}
	return &promClient{client: promv1.NewAPI(baseClient), ps: pubsub.New(0)}, nil
}

