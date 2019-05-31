package register

import (
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"

	"github.com/solo-io/glooshot/pkg/cli/options"
	"github.com/spf13/cobra"
)

func Cmd(o *options.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register",
		Short: "register the custom resources used by glooshot",
		Long:  "register the custom resources used by glooshot. This should be done once for each new cluster you use.",
		RunE: func(c *cobra.Command, args []string) error {
			// all other commands use clientsets that do not register their crds
			// for this particular command, we need to get clients that DO register their crds
			regCs := options.CreateClientset(o.Ctx, true)
			// do something trivial to register the clients
			if _, err := regCs.ReportClient().List("default", clients.ListOpts{}); err != nil {
				return err
			}
			if _, err := regCs.ExpClient().List("default", clients.ListOpts{}); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
