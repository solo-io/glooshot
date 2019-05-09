package checker

import (
	"context"
	"fmt"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/go-utils/hashutils"
)

type failureChecker struct {
	checker ExperimentChecker
}

func NewFailureChecker(checker ExperimentChecker) v1.ApiSyncDecider {
	return &failureChecker{checker: checker}
}

func (c *failureChecker) Sync(ctx context.Context, snap *v1.ApiSnapshot) error {
	ctx = contextutils.WithLogger(ctx, fmt.Sprintf("failure-checker-sync-%v", snap.Hash()))
	// we only care about started experiments
	started := startedExperiments(snap.Experiments)
	for _, exp := range started {
		exp := exp
		go func() {
			err := c.checker.MonitorExperiment(ctx, exp)
			if err != nil {
				contextutils.LoggerFrom(ctx).Errorf("monitoring experiment %v failed", exp.Metadata.Ref())
			}
		}()
	}
	return nil
}

func (c *failureChecker) ShouldSync(old, new *v1.ApiSnapshot) bool {
	updatedList := startedExperiments(new.Experiments)
	var originalList v1.ExperimentList
	if old != nil {
		originalList = startedExperiments(old.Experiments)
	}
	if len(originalList) != len(updatedList) {
		return true
	}
	for _, original := range originalList {
		updated, err := updatedList.Find(original.Metadata.Ref().Strings())
		if err != nil {
			return true
		}
		if faultsChanged(original, updated) {
			return true
		}
	}
	return false
}

func startedExperiments(list v1.ExperimentList) v1.ExperimentList {
	var started v1.ExperimentList
	list.Each(func(element *v1.Experiment) {
		if element.Result.State == v1.ExperimentResult_Started {
			started = append(started, element)
		}
	})
	return started
}

func faultsChanged(exp1, exp2 *v1.Experiment) bool {
	faults1 := hashutils.HashAll(
		exp1.Spec.FailureConditions,
		exp1.Spec.Duration,
	)
	faults2 := hashutils.HashAll(
		exp2.Spec.FailureConditions,
		exp2.Spec.Duration,
	)
	return faults1 != faults2
}
