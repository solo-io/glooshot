package premerge_contextutils

import (
	"context"

	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"
)

/*
Long form:
	contextutils.LoggerFrom(ctx).Infow("message going to file only", zap.String("cli", "info that will go to the console and file")
	contextutils.LoggerFrom(ctx).Warnw("message going to file only", zap.String("cli", "a warning that will go to the console and file"))
	contextutils.LoggerFrom(ctx).Errorw("message going to file only", zap.String("cli", "an error that will go to the console and file")

Short form with the helper:
	contextutils.CliLogInfo(ctx, "this info log should go to file and console")
	contextutils.CliLogWarn(ctx, "this warn log should go to file and console")
	contextutils.CliLogError(ctx, "this error log should go to file and console")

The helpers reduce your responsibilities from 7 decisions to 3 decisions.
So use the helpers!
*/

type cliLogLevel int

// Note that there is no Fatal log level. This is intentional.
// All errors should be surfaced up to the main entry point so that we can use
// Cobra's built-in error pipeline effectively.
const (
	cliLogLevelInfo cliLogLevel = iota + 1
	cliLogLevelWarn
	cliLogLevelError
)
const defaultCliLogKey = "cli"

func CliLogError(ctx context.Context, message string) {
	cliLog(ctx, cliLogLevelError, message, defaultCliLogKey)
}
func CliLogWarn(ctx context.Context, message string) {
	cliLog(ctx, cliLogLevelWarn, message, defaultCliLogKey)
}
func CliLogInfo(ctx context.Context, message string) {
	cliLog(ctx, cliLogLevelInfo, message, defaultCliLogKey)
}

// if we want to use a custom cliLogKey we can make this public or expose an "advanced" set of methods
func cliLog(ctx context.Context, level cliLogLevel, message, cliLogKey string) {
	log := contextutils.LoggerFrom(ctx)
	switch level {
	case cliLogLevelInfo:
		log.Infow(message, zap.String(cliLogKey, message))
	case cliLogLevelWarn:
		log.Warnw(message, zap.String(cliLogKey, message))
	case cliLogLevelError:
		log.Errorw(message, zap.String(cliLogKey, message))
	}
}
