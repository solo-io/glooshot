package inputs

import (
	"time"

	"github.com/gogo/protobuf/types"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
)

func MakeExperiment(name string) *v1.Experiment {
	experiment := v1.NewExperiment("unit-test", name)
	duration := time.Second / 2
	experiment.Spec = &v1.ExperimentSpec{
		FailureConditions: []*v1.FailureCondition{
			{
				Trigger: &v1.FailureCondition_Trigger{
					FailureTrigger: &v1.FailureCondition_Trigger_Prometheus{
						Prometheus: &v1.PrometheusTrigger{
							QueryType: &v1.PrometheusTrigger_CustomQuery{
								CustomQuery: "query1",
							},
							ThresholdValue: 50,
						},
					},
				},
			},
			{
				Trigger: &v1.FailureCondition_Trigger{
					FailureTrigger: &v1.FailureCondition_Trigger_Prometheus{
						Prometheus: &v1.PrometheusTrigger{
							QueryType: &v1.PrometheusTrigger_CustomQuery{
								CustomQuery: "query2",
							},
							ThresholdValue: 50,
						},
					},
				},
			},
		},
		Duration: &duration,
	}
	experiment.Result.TimeStarted = P(time.Now())

	experiment.Result.State = v1.ExperimentResult_Started
	return experiment
}

func P(t time.Time) *types.Timestamp {
	ts, _ := types.TimestampProto(t)
	return ts
}
