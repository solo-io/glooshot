package deleteexp

import (
	"fmt"

	"github.com/solo-io/glooshot/pkg/cli/controller"
	"github.com/solo-io/glooshot/pkg/cli/flagutils"
	"github.com/solo-io/glooshot/pkg/cli/options"
	"github.com/spf13/cobra"
)

/*------------------------------------------------------------------------------
Delete
------------------------------------------------------------------------------*/

func Cmd(o *options.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "delete a glooshot resource",
	}
	cmd.AddCommand(
		deleteExperimentCmd(o),
	)
	pflags := cmd.PersistentFlags()
	flagutils.AddMetadataFlags(pflags, &o.Metadata)
	pflags.BoolVar(&o.Delete.All, "all", false, "if set, deletes all resources in a given namespace")
	pflags.BoolVar(&o.Delete.EveryResource, "every-resource", false, "if set, deletes all resources in all namespaces")
	return cmd
}

func deleteExperimentCmd(o *options.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "experiment",
		Short:   "delete a glooshot experiment",
		Aliases: options.ExperimentAliases,
		RunE: func(c *cobra.Command, args []string) error {
			return doDeleteExperiments(o, c, args)
		},
	}
	return cmd
}

func doDeleteExperiments(o *options.Options, cmd *cobra.Command, args []string) error {
	if err := options.MetadataArgsParse(o, args, false); err != nil {
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
