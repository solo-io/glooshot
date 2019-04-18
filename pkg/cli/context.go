package cli

import (
	"context"

	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"
)

var OutputModeEnvVar = "GLOOSHOT_CLI_OUTPUT_MODE"
var ErrorMessagePreamble = "error during glooshot cli execution"
var FileLogPathElements = []string{".glooshot", "log"}

func GetInitialContextAndSetLogger(cliLogger *zap.SugaredLogger) context.Context {
	contextutils.SetFallbackLogger(cliLogger)
	loggingContext := []interface{}{"version", version.Version}
	ctx := contextutils.WithLogger(context.Background(), version.CliAppName)
	ctx = contextutils.WithLoggerValues(ctx, loggingContext...)
	return ctx
}
