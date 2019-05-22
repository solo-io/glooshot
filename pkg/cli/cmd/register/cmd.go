package register

import (
	"fmt"
	"os"

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
			// TODO(mitchdraft) - put this in a config file
			shouldRegister := os.Getenv("REGISTER_GLOOSHOT") == "1"
			if !shouldRegister {
				return fmt.Errorf("must set REGISTER_GLOOSHOT=1")
			}
			// do something trivial to register the clients
			// TODO(mitchdraft) - break this into a util, get the with-registration clients there
			if _, err := o.Clients.ReportClient().List("default", clients.ListOpts{}); err != nil {
				return err
			}
			if _, err := o.Clients.ExpClient().List("default", clients.ListOpts{}); err != nil {
				return err
			}
			return nil
		},
	}
	return cmd
}
