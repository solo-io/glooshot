package promquery_test

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	. "github.com/solo-io/glooshot/pkg/promquery"
)

var _ = Describe("Client", func() {
	It("feeds subscriptions by polling on the query interval", func() {
		poller := NewQueryPubSub(context.TODO(), newMockPromClient(), time.Millisecond)
		query1 := "query1"
		query2 := "query2"

		results1 := poller.Subscribe(query1)
		results2 := poller.Subscribe(query2)

		Eventually(func() float64 {
			select {
			case val := <-results1:
				return val
			case <-time.After(time.Second):
				return 0
			}
		}, time.Second*1).Should(Equal(float64(50)))

		Eventually(func() float64 {
			select {
			case val := <-results2:
				return val
			case <-time.After(time.Second):
				return 0
			}
		}).Should(Equal(float64(50)))
	})
})

type mockPromClient struct {
	counts map[string]model.SampleValue
	access sync.Mutex
}

func newMockPromClient() *mockPromClient {
	return &mockPromClient{counts: map[string]model.SampleValue{}}
}

func (c *mockPromClient) Query(ctx context.Context, query string, ts time.Time) (model.Value, error) {
	c.access.Lock()
	defer c.access.Unlock()
	c.counts[query]++
	return &model.Scalar{Value: c.counts[query]}, nil
}
