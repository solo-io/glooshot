package checker_test

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/solo-io/glooshot/test/inputs"

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
		reports     v1.ReportClient
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
		reports, err = v1.NewReportClient(&factory.MemoryResourceClientFactory{Cache: memory.NewInMemoryResourceCache()})
		Expect(err).NotTo(HaveOccurred())
		checker = NewChecker(queries, experiments, reports)
	})

	It("does not leak goroutines", func() {
		if os.Getenv("CI_TESTS") == "1" {
			fmt.Printf("this test is disabled in CI. to run, ensure env var `CI_TESTS` is not set to 1")
			return
		}
		startGoroutines := runtime.NumGoroutine()

		syncer := NewFailureChecker(checker)
		ctx, cancel := context.WithCancel(context.TODO())
		snap := &v1.ApiSnapshot{
			Experiments: v1.ExperimentList{inputs.MakeExperiment("aaaa"), inputs.MakeExperiment("bbbb")},
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
			Experiments: v1.ExperimentList{inputs.MakeExperiment("aaaa"), inputs.MakeExperiment("bbbb")},
		}
		should := syncer.ShouldSync(snap, snap)
		Expect(should).To(BeFalse())
	})
})
