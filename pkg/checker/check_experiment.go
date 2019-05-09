package checker

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/promquery"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/go-utils/errors"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
)

type ExperimentChecker interface {
	MonitorExperiment(ctx context.Context, experiment *v1.Experiment) error
}

type checker struct {
	promCache   promquery.QueryPubSub
	experiments v1.ExperimentClient
}

func NewChecker(queries promquery.QueryPubSub, experiments v1.ExperimentClient) *checker {
	return &checker{promCache: queries, experiments: experiments}
}

var defaultDuration = time.Minute * 10

type failureReport map[string]string

// actively track the failure conditions for an experiment
func (c *checker) MonitorExperiment(ctx context.Context, experiment *v1.Experiment) error {
	ctx = contextutils.WithLogger(ctx, "experiment-checker")
	logger := contextutils.LoggerFrom(ctx)

	// wait for the first failure of any polling
	firstFailure := make(chan failureReport, 1)

	// create a cancellable context
	// cancel all the child watches once the first result is returned
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for _, fc := range experiment.Spec.FailureConditions {
		switch trigger := fc.FailureTrigger.(type) {
		case *v1.FailureCondition_PrometeheusTrigger:
			promTrigger := trigger.PrometeheusTrigger
			ctx := ctx
			comparisonOperator := promTrigger.ComparisonOperator
			if comparisonOperator == "" {
				comparisonOperator = "<"
			}
			var queryString string
			switch query := promTrigger.QueryType.(type) {
			case *v1.PrometheusTrigger_MeshQuery_:
				return errors.Errorf("mesh query not currently supported")
			case *v1.PrometheusTrigger_CustomQuery:
				queryString = query.CustomQuery
			}
			threshold := promTrigger.ThresholdValue

			go func() {
				failure, err := c.pollUntilFailure(ctx, queryString, comparisonOperator, threshold)
				if err != nil {
					logger.Errorf("")
					return
				}

				if failure == nil {
					logger.Infof("polling cancelled")
					return
				}

				select {
				case <-ctx.Done():
					return
				case firstFailure <- failure:
				}
			}()
		}
	}

	experimentDuration := defaultDuration
	if experiment.Spec.Duration != nil {
		experimentDuration = *experiment.Spec.Duration
	}

	var report failureReport
	select {
	case <-ctx.Done():
		return nil
	case failure := <-firstFailure:
		report = failure
	case <-time.After(experimentDuration):
		// nil report means experiment passed
	}
	return c.reportResult(ctx, experiment.Metadata.Ref(), report)
}

func (c *checker) pollUntilFailure(ctx context.Context, query, comparisonOperator string, threshold float64) (failureReport, error) {
	values := c.promCache.Subscribe(query)
	defer c.promCache.Unsubscribe(query, values)
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
				return failureReport{
					"failure_type":        "value_exceeded_threshold",
					"value":               fmt.Sprintf("%v", val),
					"threshold":           fmt.Sprintf("%v", threshold),
					"comparison_operator": comparisonOperator,
				}, nil
			}
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

func (c *checker) reportResult(ctx context.Context, targetExperiment core.ResourceRef, report failureReport) error {
	experiment, err := c.experiments.Read(targetExperiment.Namespace, targetExperiment.Name, clients.ReadOpts{Ctx: ctx})
	if err != nil {
		return errors.Wrapf(err, "failed to read experiment. was it deleted since failure monitoring began?")
	}
	if report == nil {
		// success
		experiment.Result.State = v1.ExperimentResult_Succeeded
	} else {
		// failure
		experiment.Result.State = v1.ExperimentResult_Failed
		experiment.Result.FailureReport = report
	}
	experiment.Result.TimeFinished = time.Now()

	_, err = c.experiments.Write(experiment, clients.WriteOpts{
		Ctx:               ctx,
		OverwriteExisting: true,
	})

	return err
}
