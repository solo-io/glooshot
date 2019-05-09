package checker

import (
	"context"
	"github.com/solo-io/go-utils/contextutils"
	"time"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/promquery"
	"github.com/solo-io/go-utils/errors"
)

type checker struct {
	promCache   promquery.QueryPubSub
	experiments v1.ExperimentClient
}

var defaultDuration = time.Minute * 10

type comparisonOperator string

// actively track the failure conditions for an experiment
func (c *checker) MonitorExperiment(ctx context.Context, experiment *v1.Experiment) error {
	ctx = contextutils.WithLogger(ctx, "experiment-checker")
	logger := contextutils.LoggerFrom(ctx)
	for _, fc := range experiment.Spec.FailureConditions {
		switch trigger := fc.FailureTrigger.(type) {
		case *v1.FailureCondition_PrometeheusTrigger:
			promTrigger := trigger.PrometeheusTrigger
			switch query := promTrigger.QueryType.(type) {
			case *v1.FailureCondition_PrometheusTrigger_CustomQuery:
				queryString := query.CustomQuery
				experimentDuration := defaultDuration
				if experiment.Spec.Duration != nil {
					experimentDuration = *experiment.Spec.Duration
				}
				state, err := c.pollUntilThresholdExceeded(ctx, experimentDuration, queryString, promTrigger.ComparisonOperator, promTrigger.ThresholdValue)
				if err != nil {
					return err
				}

				if state == nil {
					logger.Infof("polling cancelled")
				}

				return c.updateExperiment(ctx, experiment.Metadata.Ref(), state)
			}

		}
	}
	return errors.Errorf("failure condition not implemented")
}

func (c *checker) pollUntilThresholdExceeded(ctx context.Context, duration time.Duration, query, comparisonOperator string, threshold float64) (*v1.ExperimentResult, error) {
	timeStarted := time.Now()
	values := c.promCache.Subscribe(query)
	defer c.promCache.Unsubscribe(query, values)
	timer := time.Tick(duration)
	for {
		select {
		case <-ctx.Done():
			// context cancelled, gracefully shut down
			return nil, nil
		case val, ok := <-values:
			if !ok {
				return nil, errors.Errorf("unexpected close of query subscription")
			}
			if exceededThreshold(val, threshold, comparisonOperator) {
				return &v1.ExperimentResult{
					TimeStarted: timeStarted,
					TimeElapsed: time.Now().Sub(timeStarted),
					State:       v1.ExperimentResult_Failed,
				}, nil
			}
		case <-timer:
			return &v1.ExperimentResult{
				TimeStarted: timeStarted,
				TimeElapsed: time.Now().Sub(timeStarted),
				State:       v1.ExperimentResult_Succeeded,
			}, nil
		}

	}
}

func exceededThreshold(val, threshold float64, comparisonOperator string) bool {
	switch comparisonOperator {
	case ">":
		return val > threshold
	case ">=":
		return val >= threshold
	case "<=":
		return val <= threshold

	}
	return val < threshold
}

func (c *checker) updateExperiment(ctx context.Context, duration time.Duration, query, comparisonOperator string, threshold float64) (*v1.ExperimentResult, error) {
	timeStarted := time.Now()
	values := c.promCache.Subscribe(query)
	defer c.promCache.Unsubscribe(query, values)
	timer := time.Tick(duration)
	for {
		select {
		case <-ctx.Done():
			// context cancelled, gracefully shut down
			return nil, nil
		case val, ok := <-values:
			if !ok {
				return nil, errors.Errorf("unexpected close of query subscription")
			}
			if exceededThreshold(val, threshold, comparisonOperator) {
				return &v1.ExperimentResult{
					TimeStarted: timeStarted,
					TimeElapsed: time.Now().Sub(timeStarted),
					State:       v1.ExperimentResult_Failed,
				}, nil
			}
		case <-timer:
			return &v1.ExperimentResult{
				TimeStarted: timeStarted,
				TimeElapsed: time.Now().Sub(timeStarted),
				State:       v1.ExperimentResult_Succeeded,
			}, nil
		}

	}
}