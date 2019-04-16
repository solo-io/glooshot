package cli

import (
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/solo-io/gloo/projects/gloo/cli/pkg/flagutils"
	"github.com/solo-io/gloo/projects/gloo/cli/pkg/surveyutils"
	"github.com/solo-io/glooshot/pkg/cli/printer"
	"github.com/solo-io/glooshot/pkg/gsutil"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"

	"github.com/solo-io/go-utils/contextutils"
)

/*------------------------------------------------------------------------------
Options
------------------------------------------------------------------------------*/

type Options struct {
	Clients gsutil.ClientCache
	Top     TopOptions
	// Metadata identifies a resource for commands that operate on a particular resource
	Metadata core.Metadata
	Create   CreateOptions
	Delete   DeleteOptions
	Get      GetOptions
}

type TopOptions struct {
	// ConfigFile provides default values for glooshot commands.
	// Can be overwritten by flags.
	ConfigFile string

	// Interactive indicates whether or not we are in an interactive input mode
	Interactive bool
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
		if err != nil {
			contextutils.LoggerFrom(ctx).
				Fatalw("unable to create clients for glooshot cli", zap.Error(err))
		}
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
	pflags := app.PersistentFlags()
	pflags.BoolVarP(&o.Top.Interactive, "interactive", "i", false, "use interactive mode")
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
	pflags := cmd.PersistentFlags()
	flagutils.AddMetadataFlags(pflags, &o.Metadata)
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
	if err := o.MetadataArgsParse(args); err != nil {
		return err
	}
	return o.Clients.ExpClient().Delete(o.Metadata.Namespace, o.Metadata.Name, clients.DeleteOpts{})
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
	// TODO(mitchdraft) - add an --all (namespaces) flag
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
	pflags := cmd.PersistentFlags()
	flagutils.AddMetadataFlags(pflags, &o.Metadata)
	return cmd
}

func (o *Options) doGetExperiments(cmd *cobra.Command, args []string) error {
	if err := o.MetadataArgsParse(args); err != nil {
		return err
	}
	if o.Metadata.Namespace != "" && o.Metadata.Name != "" {
		exp, err := o.Clients.ExpClient().Read(o.Metadata.Namespace, o.Metadata.Name, clients.ReadOpts{})
		if err != nil {
			return errors.Wrapf(err, "could not get experiments")
		}
		printer.Experiment(*exp)
		return nil
	}
	exps, err := o.Clients.ExpClient().List(o.Metadata.Namespace, clients.ListOpts{})
	if err != nil {
		return err
	}
	printer.PrintExperiments(exps, "")
	return nil
}

/*------------------------------------------------------------------------------
Helpers
------------------------------------------------------------------------------*/

const nameError = "name must be specified in flag (--name) or via first arg"

func (o *Options) MetadataArgsParse(args []string) error {
	// even if we are in interactive mode, we first want to check the flags and args for metadata and use those values if given
	if o.Metadata.Name == "" && len(args) > 0 {
		// name is a special parameter that can be provided in the command list
		o.Metadata.Name = args[0]
	}

	// if interactive mode, get any missing fields interactively
	if o.Top.Interactive {
		return surveyutils.EnsureMetadataSurvey(&o.Metadata)
	}

	// if not interactive mode, ensure that the required fields were provided
	if o.Metadata.Name == "" {
		return errors.Errorf(nameError)
	}
	// don't need to check namespace as is passed by a flag with a default value
	return nil
}
