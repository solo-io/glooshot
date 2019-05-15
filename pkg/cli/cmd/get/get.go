package get

import (
	"github.com/pkg/errors"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/cli/flagutils"
	"github.com/solo-io/glooshot/pkg/cli/options"
	"github.com/solo-io/glooshot/pkg/cli/printer"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/spf13/cobra"
)

func Cmd(o *options.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "get a glooshot resource",
	}
	cmd.AddCommand(
		getExperimentsCmd(o),
	)
	return cmd
}

func getExperimentsCmd(o *options.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "experiments",
		Short:   "get a glooshot experiment",
		Aliases: options.ExperimentAliases,
		RunE: func(c *cobra.Command, args []string) error {
			return doGetExperiments(o, c, args)
		},
	}
	pflags := cmd.PersistentFlags()
	flagutils.AddMetadataFlags(pflags, &o.Metadata)
	pflags.BoolVar(&o.Get.AllNamespaces, "all-namespaces", false, "if set, queries all namespaces")
	return cmd
}

func doGetExperiments(o *options.Options, cmd *cobra.Command, args []string) error {
	if err := options.MetadataArgsParse(o, args, false); err != nil {
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
		for _, ns := range options.GetNamespaces(o) {
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
