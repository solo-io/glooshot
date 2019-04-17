package main

import (
	"context"

	"github.com/solo-io/glooshot/pkg/cli"
	"github.com/solo-io/glooshot/pkg/version"
	"go.uber.org/zap"

	"github.com/solo-io/go-utils/contextutils"
)

func getInitialContext() context.Context {
	loggingContext := []interface{}{"version", version.Version}
	ctx := contextutils.WithLogger(context.Background(), version.CliAppName)
	ctx = contextutils.WithLoggerValues(ctx, loggingContext...)
	return ctx
}

func main() {
	ctx := getInitialContext()
	app := cli.App(ctx, version.Version)
	if err := app.Execute(); err != nil {
		contextutils.LoggerFrom(ctx).Fatalw("error during glooshot cli execution", zap.Error(err))
	}
}
