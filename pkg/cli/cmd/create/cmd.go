package create

import (
	"fmt"
	"io/ioutil"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/cli/options"
	"github.com/solo-io/go-utils/protoutils"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/spf13/cobra"
)

func Cmd(o *options.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a glooshot resource",
	}
	cmd.AddCommand(
		createExperimentsCmd(o),
	)
	return cmd
}

func createExperimentsCmd(o *options.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "experiment",
		Short:   "create a glooshot experiment",
		Aliases: options.ExperimentAliases,
		RunE: func(c *cobra.Command, args []string) error {
			return doCreateExperiments(o, c, args)
		},
	}
	pflags := cmd.PersistentFlags()
	pflags.StringVarP(&o.Create.CreateFile, "file", "f", "",
		"name of file containing the specification of the resource to be created")
	return cmd
}

func doCreateExperiments(o *options.Options, cmd *cobra.Command, args []string) error {
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
