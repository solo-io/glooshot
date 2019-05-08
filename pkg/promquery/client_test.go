package promquery_test

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	. "github.com/solo-io/glooshot/pkg/promquery"
	"sync"
	"time"
)

var _ = Describe("Client", func() {
	It("feeds subscriptions by polling on the query interval", func() {
		poller := NewQueryPubSub(&mockPromClient{counts: map[string]model.SampleValue{}}, time.Millisecond)
		ctx := context.TODO()
		query1 := "query1"
		errs1 := poller.BeginPolling(ctx, query1)

		query2 := "query2"
		errs2 := poller.BeginPolling(ctx, query2)

		go func() {
			defer GinkgoRecover()
			select {
			case err := <-errs1:
				Expect(err).NotTo(HaveOccurred())
			case err := <-errs2:
				Expect(err).NotTo(HaveOccurred())
			}
		}()

		results1 := poller.Subscribe(query1)
		results2 := poller.Subscribe(query2)

		Eventually(func() float64 {
			select {
			case val := <-results1:
				return val
			case <-time.After(time.Second):
				return 0
			}
		}, time.Hour).Should(Equal(float64(50)))

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
	counts map[string]model.SampleValue;
	access sync.Mutex
}

func (*mockPromClient) AlertManagers(ctx context.Context) (v1.AlertManagersResult, error) {
	panic("implement me")
}

func (*mockPromClient) CleanTombstones(ctx context.Context) error {
	panic("implement me")
}

func (*mockPromClient) Config(ctx context.Context) (v1.ConfigResult, error) {
	panic("implement me")
}

func (*mockPromClient) DeleteSeries(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) error {
	panic("implement me")
}

func (*mockPromClient) Flags(ctx context.Context) (v1.FlagsResult, error) {
	panic("implement me")
}

func (*mockPromClient) LabelValues(ctx context.Context, label string) (model.LabelValues, error) {
	panic("implement me")
}

func (c *mockPromClient) Query(ctx context.Context, query string, ts time.Time) (model.Value, error) {
	c.access.Lock()
	defer c.access.Unlock()
	c.counts[query]++
	return &model.Scalar{Value: c.counts[query]}, nil
}

func (*mockPromClient) QueryRange(ctx context.Context, query string, r v1.Range) (model.Value, error) {
	panic("implement me")
}

func (*mockPromClient) Series(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) ([]model.LabelSet, error) {
	panic("implement me")
}

func (*mockPromClient) Snapshot(ctx context.Context, skipHead bool) (v1.SnapshotResult, error) {
	panic("implement me")
}

func (*mockPromClient) Targets(ctx context.Context) (v1.TargetsResult, error) {
	panic("implement me")
}
