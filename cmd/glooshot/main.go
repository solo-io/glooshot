package main

import (
	"context"

	"github.com/solo-io/glooshot/pkg/setup"
	"go.uber.org/zap"

	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-utils/contextutils"
)

func getInitialContext() context.Context {
	loggingContext := []interface{}{"version", version.Version}
	ctx := contextutils.WithLogger(context.Background(), version.AppName)
	ctx = contextutils.WithLoggerValues(ctx, loggingContext...)
	return ctx
}

func main() {
	ctx := getInitialContext()
	if err := setup.Run(ctx, setup.GetOptions()); err != nil {
		contextutils.LoggerFrom(ctx).Fatalw("error while running glooshot", zap.Error(err))
	}
}
