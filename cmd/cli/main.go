package main

import (
	"context"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"

	"github.com/solo-io/glooshot/pkg/cli"
	"github.com/solo-io/glooshot/pkg/version"
	"go.uber.org/zap"

	"github.com/solo-io/go-utils/contextutils"
)

var outputModeEnvVar = "GLOOSHOT_CLI_OUTPUT_MODE"

func getInitialContextAndSetLogger() context.Context {
	contextutils.SetFallbackLogger(BuildCliLogger([]string{".glooshot", "log"}, outputModeEnvVar))
	loggingContext := []interface{}{"version", version.Version}
	ctx := contextutils.WithLogger(context.Background(), version.CliAppName)
	ctx = contextutils.WithLoggerValues(ctx, loggingContext...)
	return ctx
}

func main() {
	ctx := getInitialContextAndSetLogger()
	app := cli.App(ctx, version.Version)
	if err := app.Execute(); err != nil {
		contextutils.LoggerFrom(ctx).Fatalw("error during glooshot cli execution", zap.Error(err))
	}
}

////////////////////////////////////////////////////////////////////////////////
// TODO - move this to go-utils/contextutils/context_and_logging.go
////////////////////////////////////////////////////////////////////////////////
func FilePathFromHomeDir(pathElementsRelativeToHome []string) (string, error) {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	pathElements := append([]string{home}, pathElementsRelativeToHome...)
	return filepath.Join(pathElements...), nil
}

func buildCliZapCoreFile(pathElements []string, verboseMode bool) zapcore.Core {
	path, err := FilePathFromHomeDir(pathElements)
	if err != nil {
		if verboseMode {
			// we don't want to return errors just because we cannot write logs to a file
			// users can use the verbose flag to get full output to the console
			fmt.Printf("Could not open log file %s for writing: %v\n", filepath.Join(pathElements...), err)
		}
		return nil
	}

	// if we decide we want to append logs, we can do it this way:
	// (we would be required to first create the file and "rotate" it as it grows)
	//file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, os.ModeAppend)

	// for now, let's just create/overwrite the file each time
	file, err := os.Create(path)
	if err != nil {
		if verboseMode {
			// we don't want to return errors just because we cannot write logs to a file
			// users can use the verbose flag to get full output to the console
			fmt.Printf("Could not open log file %s for writing: %v\n", path, err)
		}
		return nil
	}

	passAllMessages := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return true
	})

	// apply zap's lock and WriteSyncer helpers
	fileDebug := zapcore.Lock(zapcore.AddSync(file))
	fileLoggerEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoder := zapcore.NewJSONEncoder(fileLoggerEncoderConfig)
	fileCore := zapcore.NewCore(fileEncoder, fileDebug, passAllMessages)

	return fileCore
}

func buildCliZapCoreConsoles(verboseMode bool) []zapcore.Core {

	// define error filter levels
	errorMessages := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	stdOutMessages := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl > zapcore.DebugLevel && lvl < zapcore.ErrorLevel
	})
	stdOutMessagesVerbose := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	// add locks for safe concurrency
	consoleInfo := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)

	consoleLoggerEncoderConfig := zap.NewProductionEncoderConfig()
	consoleLoggerEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	// minimize the noise for non-verbose mode
	if !verboseMode {
		consoleLoggerEncoderConfig.EncodeTime = nil
		consoleLoggerEncoderConfig.LevelKey = ""
		consoleLoggerEncoderConfig.NameKey = ""
	}
	consoleEncoder := zapcore.NewConsoleEncoder(consoleLoggerEncoderConfig)

	consoleStdoutCore := zapcore.NewCore(consoleEncoder, consoleInfo, stdOutMessages)
	if verboseMode {
		consoleStdoutCore = zapcore.NewCore(consoleEncoder, consoleInfo, stdOutMessagesVerbose)
	}
	consoleErrCore := zapcore.NewCore(consoleEncoder, consoleErrors, errorMessages)
	return []zapcore.Core{consoleStdoutCore, consoleErrCore}
}

// BuildCliLogger creates a logger that writes output to the specified filename
func BuildCliLogger(pathElements []string, outputModeEnvVar string) *zap.SugaredLogger {
	verboseMode := os.Getenv(outputModeEnvVar) == "1"
	fileCore := buildCliZapCoreFile(pathElements, verboseMode)
	consoleCores := buildCliZapCoreConsoles(verboseMode)
	allCores := consoleCores
	if fileCore != nil {
		allCores = append(allCores, fileCore)
	}
	core := zapcore.NewTee(allCores...)
	logger := zap.New(core).Sugar()
	return logger
}
