package run

import (
	"time"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/cli/flagutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	"github.com/spf13/cobra"
)

func Cmd(opts *RunExperimentOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "run",
		Aliases: []string{"r"},
		Short:   "run an experiment and wait for the result",
	}

	flagutils.AddMetadataFlags(cmd.PersistentFlags(), &opts.Metadata)

	return cmd
}

type RunExperimentOptions struct {
	Metadata core.Metadata
	Faults
}

func experimentFromOpts(opts RunExperimentOptions) (*v1.Experiment, error) {
	var (
		failureConditions []*v1.FailureCondition
		faults            []*v1.ExperimentSpec_InjectedFault
		duration          *time.Duration
	)
	return &v1.Experiment{
		Metadata: opts.Metadata,
		Spec: &v1.ExperimentSpec{
			Faults:            faults,
			FailureConditions: failureConditions,
			Duration:          duration,
			TargetMesh:        targetMesh,
		},
	}, nil
}
