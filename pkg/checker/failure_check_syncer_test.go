package checker_test

import (
	"context"
	"runtime"
	"time"

	"github.com/gogo/protobuf/types"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/promquery"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/memory"

	. "github.com/solo-io/glooshot/pkg/checker"
)

var _ = Describe("FailureCheckSyncer", func() {

	var (
		experiments v1.ExperimentClient
		checker     ExperimentChecker
		prom        *mockPromClient
	)
	BeforeEach(func() {
		var err error
		prom = newMockPromClient()
		prom.nextValue = func(query string) model.SampleValue {
			return 100
		}
		queries := promquery.NewQueryPubSub(context.TODO(), prom, time.Millisecond)
		experiments, err = v1.NewExperimentClient(&factory.MemoryResourceClientFactory{Cache: memory.NewInMemoryResourceCache()})
		Expect(err).NotTo(HaveOccurred())
		checker = NewChecker(queries, experiments)
	})

	It("does not leak goroutines", func() {
		startGoroutines := runtime.NumGoroutine()

		syncer := NewFailureChecker(checker)
		ctx, cancel := context.WithCancel(context.TODO())
		snap := &v1.ApiSnapshot{
			Experiments: v1.ExperimentList{makeExperiment("aaaa"), makeExperiment("bbbb")},
		}
		err := syncer.Sync(ctx, snap)
		Expect(err).NotTo(HaveOccurred())

		cancel()

		ctx, cancel = context.WithCancel(context.TODO())
		err = syncer.Sync(ctx, snap)
		Expect(err).NotTo(HaveOccurred())

		cancel()

		ctx, cancel = context.WithCancel(context.TODO())
		err = syncer.Sync(ctx, snap)
		Expect(err).NotTo(HaveOccurred())

		cancel()

		Eventually(func() int {
			return runtime.NumGoroutine()
		}, time.Second*15).Should(Equal(startGoroutines))
	})

	It("should not sync if the old and new snapshots are equal", func() {
		syncer := NewFailureChecker(checker)
		snap := &v1.ApiSnapshot{
			Experiments: v1.ExperimentList{makeExperiment("aaaa"), makeExperiment("bbbb")},
		}
		should := syncer.ShouldSync(snap, snap)
		Expect(should).To(BeFalse())
	})
})

func makeExperiment(name string) *v1.Experiment {
	experiment := v1.NewExperiment("unit-test", name)
	duration := time.Second / 2
	experiment.Spec = &v1.ExperimentSpec{
		FailureConditions: []*v1.FailureCondition{
			{
				FailureTrigger: &v1.FailureCondition_PrometheusTrigger{
					PrometheusTrigger: &v1.PrometheusTrigger{
						QueryType: &v1.PrometheusTrigger_CustomQuery{
							CustomQuery: "query1",
						},
						ThresholdValue: 50,
					},
				},
			},
			{
				FailureTrigger: &v1.FailureCondition_PrometheusTrigger{
					PrometheusTrigger: &v1.PrometheusTrigger{
						QueryType: &v1.PrometheusTrigger_CustomQuery{
							CustomQuery: "query2",
						},
						ThresholdValue: 50,
					},
				},
			},
		},
		Duration: &duration,
	}
	experiment.Result.TimeStarted, _ = types.TimestampProto(time.Now())

	experiment.Result.State = v1.ExperimentResult_Started
	return experiment
}
