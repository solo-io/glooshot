package data

import (
	"time"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	sgv1 "github.com/solo-io/supergloo/pkg/api/v1"
)

// some values to work with
var duration1 = time.Hour
var duration2 = time.Second

var basicAbortFault = &sgv1.FaultInjection{
	FaultInjectionType: &sgv1.FaultInjection_Abort_{
		Abort: &sgv1.FaultInjection_Abort{
			ErrorType: &sgv1.FaultInjection_Abort_HttpStatus{
				HttpStatus: 404,
			},
		},
	},
	Percentage: 50.0,
}
var basicDelayFault = &sgv1.FaultInjection{
	FaultInjectionType: &sgv1.FaultInjection_Delay_{
		Delay: &sgv1.FaultInjection_Delay{
			Duration:  duration2,
			DelayType: sgv1.FaultInjection_Delay_FIXED,
		},
	},
	Percentage: 50.0,
}

var destination1 = &core.ResourceRef{"name1", "default"}
var destination2 = &core.ResourceRef{"name2", "default"}

var basicFailureCondition = &v1.FailureCondition{
	FailureTrigger: &v1.FailureCondition_PrometheusTrigger{
		PrometheusTrigger: &v1.PrometheusTrigger{
			QueryType: &v1.PrometheusTrigger_CustomQuery{
				CustomQuery: "cpu percent",
			},
			ThresholdValue:     10,
			ComparisonOperator: "",
		},
	},
}

// some composition functions

func GetBasicExperiment(namespace, name string) *v1.Experiment {
	return &v1.Experiment{
		Metadata: core.Metadata{
			Name:      name,
			Namespace: namespace,
		},
		Spec: &v1.ExperimentSpec{
			Faults: []*v1.ExperimentSpec_InjectedFault{{
				OriginServices:      []*core.ResourceRef{destination1},
				DestinationServices: []*core.ResourceRef{destination2},
				Fault:               basicAbortFault,
			}},
			FailureConditions: []*v1.FailureCondition{basicFailureCondition},
			Duration:          &duration1,
			TargetMesh: &core.ResourceRef{
				Name:      "basicmesh",
				Namespace: namespace,
			},
		},
	}
}

func GetBasicExperimentAbort(namespace, name string) *v1.Experiment {
	return GetBasicExperiment(namespace, name)
}

func GetBasicExperimentDelay(namespace, name string) *v1.Experiment {
	exp := GetBasicExperiment(namespace, name)
	exp.Spec.Faults = []*v1.ExperimentSpec_InjectedFault{
		oneToOneFault(destination1, destination2, basicDelayFault),
	}
	return exp
}

func oneToOneFault(origin, destination *core.ResourceRef, fault *sgv1.FaultInjection) *v1.ExperimentSpec_InjectedFault {
	return &v1.ExperimentSpec_InjectedFault{
		OriginServices:      []*core.ResourceRef{origin},
		DestinationServices: []*core.ResourceRef{destination},
		Fault:               fault,
	}
}
