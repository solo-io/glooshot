package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/solo-io/glooshot/pkg/gsutil"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/solo-io/solo-kit/pkg/api/v1/clients"

	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"

	gloov1 "github.com/solo-io/gloo/projects/gloo/pkg/api/v1"
	"github.com/solo-io/gloo/projects/gloo/pkg/api/v1/plugins/faultinjection"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/go-utils/contextutils"
)

/*------------------------------------------------------------------------------
Options
------------------------------------------------------------------------------*/

type Options struct {
	Clients gsutil.ClientCache
	Top     TopOptions
	Create  CreateOptions
	Delete  DeleteOptions
	Get     GetOptions
}

type TopOptions struct {
	// ConfigFile provides default values for glooshot commands.
	// Can be overwritten by flags.
	ConfigFile string
}

type CreateOptions struct {
	// CreateFile contains the glooshot api resource that should be created
	CreateFile string
}

type DeleteOptions struct {
	Experiment core.ResourceRef
}

type GetOptions struct {
	Experiment core.ResourceRef
}

const defaultConfigFileLocation = "~/.glooshot/config.yaml"

func initialOptions(ctx context.Context, registerCrds bool) Options {
	return Options{
		Clients: gsutil.NewClientCache(ctx, registerCrds, cliClientErrorHandler(ctx)),
		Top: TopOptions{
			ConfigFile: defaultConfigFileLocation,
		},
		Create: CreateOptions{},
	}
}

func cliClientErrorHandler(ctx context.Context) func(error) {
	return func(err error) {
		contextutils.LoggerFrom(ctx).
			Fatalw("unable to create clients for glooshot cli", zap.Error(err))
	}
}

/*------------------------------------------------------------------------------
Root
------------------------------------------------------------------------------*/

func App(ctx context.Context, version string) *cobra.Command {
	app := &cobra.Command{
		Use:     "glooshot",
		Short:   "CLI for glooshot",
		Version: version,
	}

	register := os.Getenv("REGISTER_GLOOSHOT") == "1"
	o := initialOptions(ctx, register)

	app.AddCommand(
		o.createCmd(),
		o.deleteCmd(),
		o.getCmd(),
		completionCmd(),
	)
	return app
}

/*------------------------------------------------------------------------------
Create
------------------------------------------------------------------------------*/

func (o *Options) createCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a glooshot resource",
	}
	cmd.AddCommand(
		o.createExperimentsCmd(),
	)
	return cmd
}
func (o *Options) createExperimentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experiment",
		Short: "create a glooshot experiment",
		RunE: func(c *cobra.Command, args []string) error {
			return o.doCreateExperiments(c, args)
		},
	}
	return cmd
}
func (o *Options) doCreateExperiments(cmd *cobra.Command, args []string) error {
	panic("TODO")
	return nil
}

/*------------------------------------------------------------------------------
Delete
------------------------------------------------------------------------------*/

func (o *Options) deleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete a glooshot resource",
	}
	cmd.AddCommand(
		o.deleteExperimentCmd(),
	)
	return cmd
}
func (o *Options) deleteExperimentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experiment",
		Short: "delete a glooshot experiment",
		RunE: func(c *cobra.Command, args []string) error {
			return o.doDeleteExperiments(c, args)
		},
	}
	return cmd
}
func (o *Options) doDeleteExperiments(cmd *cobra.Command, args []string) error {
	panic("TODO")
	return nil
}

/*------------------------------------------------------------------------------
Get
------------------------------------------------------------------------------*/

func (o *Options) getCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get a glooshot resource",
	}
	cmd.AddCommand(
		o.getExperimentsCmd(),
	)
	return cmd
}
func (o *Options) getExperimentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experiments",
		Short: "get a glooshot experiment",
		RunE: func(c *cobra.Command, args []string) error {
			return o.doGetExperiments(c, args)
		},
	}
	return cmd
}

func (o *Options) doGetExperiments(cmd *cobra.Command, args []string) error {
	panic("TODO")
	return nil
}

func Run(ctx context.Context) error {
	var mode string
	var namespace string
	var name string
	flag.StringVar(&mode, "mode", "", "specify a mode (temporary helper cmd, will be removed)")
	flag.StringVar(&namespace, "namespace", "default", "namespace of experiment")
	flag.StringVar(&name, "name", "", "name of experiment")
	flag.Parse()

	if name == "" {
		return fmt.Errorf("must provide an experiment namme")
	}

	client, err := gsutil.GetExperimentClient(ctx, false)
	if err != nil {
		return err
	}

	minute := time.Minute
	delayTime := time.Second
	exp := &v1.Experiment{
		Metadata: core.Metadata{
			Namespace: namespace,
			Name:      name,
		},
	}
	faults1 := []*v1.ExperimentSpec_InjectedFault{{
		Service: &gloov1.Destination{
			Upstream: core.ResourceRef{
				Name:      "todo",
				Namespace: "default",
			},
		},
		Fault: &faultinjection.RouteFaults{
			Delay: &faultinjection.RouteDelay{
				Percentage: 100,
				FixedDelay: &delayTime,
			},
		},
	}}
	faults2 := []*v1.ExperimentSpec_InjectedFault{{
		Service: &gloov1.Destination{
			Upstream: core.ResourceRef{
				Name:      "todo",
				Namespace: "default",
			},
		},
		Fault: &faultinjection.RouteFaults{
			Abort: &faultinjection.RouteAbort{
				Percentage: 100,
				HttpStatus: 404,
			},
		},
	}}
	stop1 := &v1.StopCondition{
		Duration: &minute,
		Metric: []*v1.MetricThreshold{
			{
				MetricName: "dinner",
				Value:      1800,
			}},
	}
	switch mode {
	case "a":
		exp.Spec = &v1.ExperimentSpec{
			Faults:        faults1,
			StopCondition: stop1,
		}
	case "b":
		exp.Spec = &v1.ExperimentSpec{
			Faults:        faults2,
			StopCondition: stop1,
		}
	default:
	}
	fmt.Println("attempting to write")
	_, err = client.Write(exp, clients.WriteOpts{})
	return err
}
