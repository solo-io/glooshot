package options

import (
	"context"

	"github.com/pkg/errors"
	"github.com/solo-io/gloo/projects/gloo/cli/pkg/surveyutils"

	"github.com/solo-io/glooshot/pkg/cli/gsutil"
	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var ExperimentAliases = []string{"experiment", "experiments", "exp"}

/*------------------------------------------------------------------------------
Options
------------------------------------------------------------------------------*/

type Options struct {
	Ctx     context.Context
	Clients gsutil.ClientCache
	Top     TopOptions
	// Metadata identifies a resource for commands that operate on a particular resource
	Metadata core.Metadata
	Create   CreateOptions
	Delete   DeleteOptions
	Get      GetOptions
	Init     Init
	cache    optionsCache
}

type TopOptions struct {
	// ConfigFile provides default values for glooshot commands.
	// Can be overwritten by flags.
	// TODO(mitchdraft) -add config file
	ConfigFile string

	// Interactive indicates whether or not we are in an interactive input mode
	Interactive bool
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

type Init struct {
	HelmChartOverride string
	HelmValues        string
	InstallNamespace  string
	ReleaseVersion    string
	DryRun            bool
}

type optionsCache struct {
	nsList []string
}

func GetNamespaces(o *Options) []string {
	if o.cache.nsList == nil {
		nsList, err := o.Clients.KubeClient().CoreV1().Namespaces().List(metav1.ListOptions{})
		if err != nil {
			contextutils.LoggerFrom(o.Ctx).Fatalw("unable to list namespaces", zap.Error(err))
		}
		for _, ns := range nsList.Items {
			o.cache.nsList = append(o.cache.nsList, ns.Name)
		}
	}
	return o.cache.nsList
}

const defaultConfigFileLocation = "~/.glooshot/config.yaml"

func InitialOptions(ctx context.Context, registerCrds bool) Options {
	return Options{
		Ctx:     ctx,
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

const nameError = "name must be specified in flag (--name) or via first arg"

func MetadataArgsParse(o *Options, args []string, nameRequired bool) error {
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
