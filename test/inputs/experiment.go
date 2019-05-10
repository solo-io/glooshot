package inputs

import (
	"time"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
)

func MakeExperiment(name string) *v1.Experiment {
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
	experiment.Result.TimeStarted = time.Now()

	experiment.Result.State = v1.ExperimentResult_Started
	return experiment
}
