package main

import (
	"github.com/solo-io/glooshot/pkg/cli"
	"github.com/solo-io/glooshot/pkg/version"
	"go.uber.org/zap"

	"github.com/solo-io/go-utils/contextutils"
)

func main() {
	cliLogger := cli.BuildCliLogger([]string{".glooshot", "log"}, cli.OutputModeEnvVar)
	ctx := cli.GetInitialContextAndSetLogger(cliLogger)
	app := cli.App(ctx, version.Version)
	if err := app.Execute(); err != nil {
		contextutils.LoggerFrom(ctx).Fatalw(cli.ErrorMessagePreamble, zap.Error(err))
	}
}
