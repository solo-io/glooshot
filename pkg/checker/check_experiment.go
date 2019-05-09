package checker

import (
	"context"
	"time"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/promquery"
	"github.com/solo-io/go-utils/errors"
)

type checker struct {
	promCache *promquery.QueryPubSub
}

// actively track the failure conditions for an experiment
func (c *checker) MonitorExperiment(ctx context.Context, experiment *v1.Experiment) error {
	for _, fc := range experiment.Spec.FailureConditions {
		switch trigger := fc.FailureTrigger.(type) {
		case *v1.FailureCondition_PrometeheusTrigger:
			promTrigger := trigger.PrometeheusTrigger
			switch query := promTrigger.QueryType.(type) {
			case *v1.FailureCondition_PrometheusTrigger_CustomQuery:
				queryString := query.CustomQuery
				state, err := c.pollUntilThresholdExceeded(ctx, experiment.Spec.Duration, queryString, promTrigger.ComparisonOperator, promTrigger.ThresholdValue)
				if err != nil {
					return err
				}

				return updateExperiment(ctx, experiment.Metadata.Ref(), state)
			}

		}
	}
	return errors.Errorf("failure condition not implemented")
}

func (c *checker) pollUntilThresholdExceeded(ctx context.Context) (*v1.ExperimentResult, error) {
	timeStarted := time.Now()
	values := c.promCache.Subscribe(query)
	ticker := time.Tick()
	for {
		select {
		case <-ctx.Done():
			// context cancelled, gracefully shut down
			return nil, nil
		case val, ok := <-values:
			if !ok {
				return nil, errors.Errorf("unexpected close of query subscription")
			}
			if exceededThreshold(val, comparisonOperator, threshold) {
				return &v1.ExperimentResult{
					TimeStarted: timeStarted,
					TimeElapsed: time.Now().Sub(timeStarted),
					State: v1.ExperimentResult_Failed,
				}, nil
			}
		}

	}
}
