package checker_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/prometheus/common/model"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/promquery"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/memory"

	. "github.com/solo-io/glooshot/pkg/checker"
)

var _ = Describe("CheckExperiment", func() {
	var (
		experiments v1.ExperimentClient
		checker     ExperimentChecker
		prom        *mockPromClient
		q1, q2      = "physics1", "astronomy2"
	)
	BeforeEach(func() {
		var err error
		prom = newMockPromClient()
		queries := promquery.NewQueryPubSub(context.TODO(), prom, time.Millisecond)
		experiments, err = v1.NewExperimentClient(&factory.MemoryResourceClientFactory{Cache: memory.NewInMemoryResourceCache()})
		Expect(err).NotTo(HaveOccurred())
		checker = NewChecker(queries, experiments)
	})

	Context("failure condition met", func() {
		It("sets the experiment state to failed", func() {
			var next model.SampleValue = 100
			prom.nextValue = func(query string) model.SampleValue {
				switch query {
				// q1 will hit threshold
				case q2:
					next--
					return next
				}
				return 100
			}
			experiment := v1.NewExperiment("albert", "einstein")
			experiment.Spec = &v1.ExperimentSpec{
				FailureConditions: []*v1.FailureCondition{
					{
						FailureTrigger: &v1.FailureCondition_PrometeheusTrigger{
							PrometeheusTrigger: &v1.PrometheusTrigger{
								QueryType: &v1.PrometheusTrigger_CustomQuery{
									CustomQuery: q1,
								},
								ThresholdValue: 50,
							},
						},
					},
					{
						FailureTrigger: &v1.FailureCondition_PrometeheusTrigger{
							PrometeheusTrigger: &v1.PrometheusTrigger{
								QueryType: &v1.PrometheusTrigger_CustomQuery{
									CustomQuery: q2,
								},
								ThresholdValue: 50,
							},
						},
					},
				},
			}

			// load the experiment into storage
			experiment, err := experiments.Write(experiment, clients.WriteOpts{})
			Expect(err).NotTo(HaveOccurred())

			go func() {
				defer GinkgoRecover()
				err := checker.MonitorExperiment(context.TODO(), experiment)
				Expect(err).NotTo(HaveOccurred())
			}()

			// read the experiemnt and check the result
			Eventually(func() (*v1.ExperimentResult, error) {
				exp, err := experiments.Read(experiment.Metadata.Namespace, experiment.Metadata.Name, clients.ReadOpts{})
				if err != nil {
					return nil, err
				}
				exp.Result.TimeStarted = time.Time{}
				exp.Result.TimeFinished = time.Time{}
				return &exp.Result, nil
			}, time.Second*3).Should(Equal(&v1.ExperimentResult{
				State: v1.ExperimentResult_Failed,
				FailureReport: map[string]string{
					"failure_type":        "value_exceeded_threshold",
					"value":               "49",
					"threshold":           "50",
					"comparison_operator": "<",
				},
			}))
		})
	})
	Context("failure condition met", func() {
		It("sets the experiment state to failed", func() {
			prom.nextValue = func(query string) model.SampleValue {
				// none will hit threshold
				return 100
			}
			experiment := v1.NewExperiment("albert", "einstein")
			duration := time.Second / 2
			experiment.Spec = &v1.ExperimentSpec{
				FailureConditions: []*v1.FailureCondition{
					{
						FailureTrigger: &v1.FailureCondition_PrometeheusTrigger{
							PrometeheusTrigger: &v1.PrometheusTrigger{
								QueryType: &v1.PrometheusTrigger_CustomQuery{
									CustomQuery: q1,
								},
								ThresholdValue: 50,
							},
						},
					},
					{
						FailureTrigger: &v1.FailureCondition_PrometeheusTrigger{
							PrometeheusTrigger: &v1.PrometheusTrigger{
								QueryType: &v1.PrometheusTrigger_CustomQuery{
									CustomQuery: q2,
								},
								ThresholdValue: 50,
							},
						},
					},
				},
				Duration: &duration,
			}

			// load the experiment into storage
			experiment, err := experiments.Write(experiment, clients.WriteOpts{})
			Expect(err).NotTo(HaveOccurred())

			go func() {
				defer GinkgoRecover()
				err := checker.MonitorExperiment(context.TODO(), experiment)
				Expect(err).NotTo(HaveOccurred())
			}()

			// read the experiemnt and check the result
			Eventually(func() (*v1.ExperimentResult, error) {
				exp, err := experiments.Read(experiment.Metadata.Namespace, experiment.Metadata.Name, clients.ReadOpts{})
				if err != nil {
					return nil, err
				}
				exp.Result.TimeStarted = time.Time{}
				exp.Result.TimeFinished = time.Time{}
				return &exp.Result, nil
			}, time.Second*3).Should(Equal(&v1.ExperimentResult{
				State: v1.ExperimentResult_Succeeded,
			}))
		})
	})
})

type mockPromClient struct {
	nextValue func(query string) model.SampleValue
}

func newMockPromClient() *mockPromClient {
	return &mockPromClient{}
}

func (c *mockPromClient) Query(ctx context.Context, query string, ts time.Time) (model.Value, error) {
	return &model.Scalar{Value: c.nextValue(query)}, nil
}
