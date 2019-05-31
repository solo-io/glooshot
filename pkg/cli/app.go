package cli

import (
	"context"

	"github.com/solo-io/glooshot/pkg/cli/cmd/register"

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
	o := options.InitialOptions(ctx)
	app := &cobra.Command{
		Use:     "glooshot",
		Short:   "CLI for glooshot",
		Version: version,
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	app.AddCommand(
		register.Cmd(&o),
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
