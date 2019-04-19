package premerge_contextutils

import (
	"context"

	"github.com/solo-io/go-utils/contextutils"
	"go.uber.org/zap"
)

type cliLogLevel int

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
