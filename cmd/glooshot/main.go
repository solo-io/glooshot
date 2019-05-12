package main

import (
	"context"

	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-utils/contextutils"

	"github.com/solo-io/glooshot/pkg/setup"
)

func getInitialContext() context.Context {
	loggingContext := []interface{}{"version", version.Version}
	ctx := contextutils.WithLogger(context.Background(), version.AppName)
	ctx = contextutils.WithLoggerValues(ctx, loggingContext...)
	return ctx
}

func main() {
	ctx := getInitialContext()
	contextutils.LoggerFrom(ctx).Fatal(setup.Run(ctx, setup.GetOptions()))
}
