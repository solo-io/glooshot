package cli

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	premerge_contextutils "github.com/solo-io/glooshot/pkg/cli/premerge-contextutils"

	"github.com/solo-io/glooshot/pkg/controller"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/go-utils/protoutils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	ctx     context.Context
	Clients gsutil.ClientCache
	Top     TopOptions
	// Metadata identifies a resource for commands that operate on a particular resource
	Metadata core.Metadata
	Create   CreateOptions
	Delete   DeleteOptions
	Get      GetOptions
	cache    optionsCache
}

type TopOptions struct {
	// ConfigFile provides default values for glooshot commands.
	// Can be overwritten by flags.
	// TODO(mitchdraft) -add config file
	ConfigFile string

	// Interactive indicates whether or not we are in an interactive input mode
	Interactive bool
	// TODO - REMOVE
	Temp bool
}

type CreateOptions struct {
	// CreateFile contains the glooshot api resource that should be created
	CreateFile string
}

type DeleteOptions struct {
	// All indicates that all resources in the given namespace should be deleted
	All bool
	// EveryResource indicates that all resources in all namespaces should be deleted
	EveryResource bool
}

type GetOptions struct {
	AllNamespaces bool
}

type optionsCache struct {
	nsList []string
}

func (o *Options) GetNamespaces() []string {
	if o.cache.nsList == nil {
		nsList, err := o.Clients.KubeClient().CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			contextutils.LoggerFrom(o.ctx).Fatalw("unable to list namespaces", zap.Error(err))
		}
		for _, ns := range nsList.Items {
			o.cache.nsList = append(o.cache.nsList, ns.Name)
		}
	}
	return o.cache.nsList
}

const defaultConfigFileLocation = "~/.glooshot/config.yaml"

func initialOptions(ctx context.Context, registerCrds bool) Options {
	return Options{
		ctx:     ctx,
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
	// TODO(mitchdraft) - put this in a config file
	register := os.Getenv("REGISTER_GLOOSHOT") == "1"
	o := initialOptions(ctx, register)
	app := &cobra.Command{
		Use:     "glooshot",
		Short:   "CLI for glooshot",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.Top.Temp {
				// Trigger some warnings, this will be removed
				premerge_contextutils.CliLogInfo(ctx, "this info log should go to file and console")
				premerge_contextutils.CliLogWarn(ctx, "this warn log should go to file and console")
				premerge_contextutils.CliLogError(ctx, "this error log should go to file and console")
			}
			return nil
		},
	}

	app.AddCommand(
		o.createCmd(),
		o.deleteCmd(),
		o.getCmd(),
		completionCmd(),
	)
	pflags := app.PersistentFlags()
	pflags.BoolVarP(&o.Top.Interactive, "interactive", "i", false, "use interactive mode")
	pflags.BoolVarP(&o.Top.Temp, "temp", "t", false, "this is a temp flag that will be removed after refactor")
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
		Use:     "experiment",
		Short:   "create a glooshot experiment",
		Aliases: experimentAliases,
		RunE: func(c *cobra.Command, args []string) error {
			return o.doCreateExperiments(c, args)
		},
	}
	pflags := cmd.PersistentFlags()
	pflags.StringVarP(&o.Create.CreateFile, "file", "f", "",
		"name of file containing the specification of the resource to be created")
	return cmd
}

func (o *Options) doCreateExperiments(cmd *cobra.Command, args []string) error {
	if o.Create.CreateFile == "" {
		return fmt.Errorf("no experiment specification file provided")
	}
	content, err := ioutil.ReadFile(o.Create.CreateFile)
	if err != nil {
		return err
	}
	exp := &v1.Experiment{}
	if err := protoutils.UnmarshalYaml(content, exp); err != nil {
		return err
	}
	_, err = o.Clients.ExpClient().Write(exp, clients.WriteOpts{OverwriteExisting: false})
	if err != nil {
		return err
	}
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
	pflags.BoolVar(&o.Delete.All, "all", false, "if set, deletes all resources in a given namespace")
	pflags.BoolVar(&o.Delete.EveryResource, "every-resource", false, "if set, deletes all resources in all namespaces")
	return cmd
}

func (o *Options) deleteExperimentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "experiment",
		Short:   "delete a glooshot experiment",
		Aliases: experimentAliases,
		RunE: func(c *cobra.Command, args []string) error {
			return o.doDeleteExperiments(c, args)
		},
	}
	return cmd
}

func (o *Options) doDeleteExperiments(cmd *cobra.Command, args []string) error {
	if err := o.MetadataArgsParse(args, false); err != nil {
		return err
	}
	ctrl := controller.From(o.Clients)
	if o.Delete.EveryResource {
		return ctrl.DeleteAllExperiments()
	}
	if o.Delete.All {
		if o.Metadata.Namespace == "" {
			return fmt.Errorf("please provide a namespace when using the --all flag")
		}
		return ctrl.DeleteExperiments(o.Metadata.Namespace)
	}
	if o.Metadata.Namespace == "" || o.Metadata.Name == "" {
		return fmt.Errorf("please provide a name and namespace")
	}
	return controller.From(o.Clients).DeleteExperiment(o.Metadata.Namespace, o.Metadata.Name)
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
		Use:     "experiments",
		Short:   "get a glooshot experiment",
		Aliases: experimentAliases,
		RunE: func(c *cobra.Command, args []string) error {
			return o.doGetExperiments(c, args)
		},
	}
	pflags := cmd.PersistentFlags()
	flagutils.AddMetadataFlags(pflags, &o.Metadata)
	pflags.BoolVar(&o.Get.AllNamespaces, "all-namespaces", false, "if set, queries all namespaces")
	return cmd
}

func (o *Options) doGetExperiments(cmd *cobra.Command, args []string) error {
	if err := o.MetadataArgsParse(args, false); err != nil {
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
	exps := []*v1.Experiment{}
	if o.Get.AllNamespaces {
		for _, ns := range o.GetNamespaces() {
			nsExps, err := o.Clients.ExpClient().List(ns, clients.ListOpts{})
			if err != nil {
				return err
			}
			exps = append(exps, nsExps...)
		}

	} else {
		var err error
		exps, err = o.Clients.ExpClient().List(o.Metadata.Namespace, clients.ListOpts{})
		if err != nil {
			return err
		}
	}
	printer.PrintExperiments(exps, "")
	return nil
}

/*------------------------------------------------------------------------------
Helpers
------------------------------------------------------------------------------*/

const nameError = "name must be specified in flag (--name) or via first arg"

func (o *Options) MetadataArgsParse(args []string, nameRequired bool) error {
	// even if we are in interactive mode, we first want to check the flags and args for metadata and use those values if given
	if o.Metadata.Name == "" && len(args) > 0 {
		// name is a special parameter that can be provided in the command list
		o.Metadata.Name = args[0]
	}

	// if interactive mode, get any missing fields interactively
	if o.Top.Interactive {
		// TODO(mitchdraft) - make an variant of this util that takes an optional list for autocompletion help
		return surveyutils.EnsureMetadataSurvey(&o.Metadata)
	}

	// if not interactive mode, ensure that the required fields were provided
	if nameRequired && o.Metadata.Name == "" {
		return errors.Errorf(nameError)
	}
	// don't need to check namespace as is passed by a flag with a default value
	return nil
}

var experimentAliases = []string{"experiment", "experiments", "experiment"}
