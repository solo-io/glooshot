package starter

import (
	"context"
	"fmt"
	"time"

	"github.com/solo-io/go-utils/kubeutils"

	"github.com/gogo/protobuf/types"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/utils"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"go.uber.org/multierr"
)

// simple syncer, marks experiments as started
type experimentStarter struct {
	experiments v1.ExperimentClient
}

func NewExperimentStarter(experiments v1.ExperimentClient) v1.ApiSyncer {
	return &experimentStarter{experiments: experiments}
}

func (s *experimentStarter) Sync(ctx context.Context, snap *v1.ApiSnapshot) error {
	ctx = contextutils.WithLogger(ctx, fmt.Sprintf("experiment-starter-sync-%v", snap.Hash()))
	logger := contextutils.LoggerFrom(ctx)
	logger.Infof("begin sync %v", snap.Hash())
	defer logger.Infof("end sync %v", snap.Hash())
	logger.Debugf("full snapshot: %v", snap)

	pending := utils.ExperimentsWithState(snap.Experiments, v1.ExperimentResult_Pending)
	now, err := types.TimestampProto(time.Now())
	if err != nil {
		logger.Panic("failed converting time.Now() to proto")
	}

	var errs error
	pending.Each(func(experimentToStart *v1.Experiment) {
		if err := s.writeAsStarted(ctx, experimentToStart, now); err != nil {
			errs = multierr.Append(errs, err)
		}
	})

	return errs
}

func (s *experimentStarter) ShouldSync(old, new *v1.ApiSnapshot) bool {
	return len(utils.ExperimentsWithState(new.Experiments, v1.ExperimentResult_Pending)) > 0
}

func (s *experimentStarter) writeAsStarted(ctx context.Context, experimentToStart *v1.Experiment, now *types.Timestamp) error {
	experimentToStart.Result.TimeStarted = now
	experimentToStart.Result.State = v1.ExperimentResult_Started
	if err := validateOrGenerateFailureConditionNames(experimentToStart); err != nil {
		return err
	}
	_, err := s.experiments.Write(experimentToStart, clients.WriteOpts{Ctx: ctx, OverwriteExisting: true})
	return err
}

func validateOrGenerateFailureConditionNames(exp *v1.Experiment) error {
	nameMap := make(map[string]bool)
	for i, fc := range exp.Spec.FailureConditions {
		if fc.Name == "" {
			fc.Name = kubeutils.SanitizeName(fmt.Sprintf("%v-%v", i, time.Now().UnixNano()))
		}
		if _, exists := nameMap[fc.Name]; exists {
			return fmt.Errorf("duplicate failure condition names are not allowed, found multiple with name %v", fc.Name)
		}
		nameMap[fc.Name] = true
	}
	return nil
}
