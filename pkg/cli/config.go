package cli

import (
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/go-utils/clicore"
)

var GlooshotConfig = clicore.CommandConfig{
	Command:             App,
	Version:             version.Version,
	FileLogPathElements: FileLogPathElements,
	OutputModeEnvVar:    OutputModeEnvVar,
	RootErrorMessage:    ErrorMessagePreamble,
	LoggingContext:      []interface{}{"version", version.Version},
}
