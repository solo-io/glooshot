package cli

import (
	"context"
	"os"

	"github.com/solo-io/glooshot/pkg/cli/cmd/create"
	"github.com/solo-io/glooshot/pkg/cli/cmd/deleteexp"
	"github.com/solo-io/glooshot/pkg/cli/cmd/get"
	"github.com/solo-io/glooshot/pkg/cli/cmd/initexp"

	"github.com/solo-io/glooshot/pkg/cli/options"

	"github.com/spf13/cobra"
)

/*------------------------------------------------------------------------------
Root
------------------------------------------------------------------------------*/

func App(ctx context.Context, version string) *cobra.Command {
	// TODO(mitchdraft) - put this in a config file
	register := os.Getenv("REGISTER_GLOOSHOT") == "1"
	o := options.InitialOptions(ctx, register)
	app := &cobra.Command{
		Use:     "glooshot",
		Short:   "CLI for glooshot",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	app.AddCommand(
		create.Cmd(&o),
		deleteexp.Cmd(&o),
		get.Cmd(&o),
		initexp.Cmd(&o),
		completionCmd(),
	)
	pflags := app.PersistentFlags()
	pflags.BoolVarP(&o.Top.Interactive, "interactive", "i", false, "use interactive mode")
	return app
}
