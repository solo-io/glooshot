package utils

import (
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
)

func ExperimentsWithState(list v1.ExperimentList, state v1.ExperimentResult_State) v1.ExperimentList {
	var started v1.ExperimentList
	list.Each(func(element *v1.Experiment) {
		if element.Result.State == state {
			started = append(started, element)
		}
	})
	return started
}
