package cli

import (
	clilog "github.com/solo-io/glooshot/pkg/pregoutils-clilog"
	"github.com/solo-io/glooshot/pkg/version"
)

var GlooshotConfig = clilog.CommandConfig{
	Command:             App,
	Version:             version.Version,
	FileLogPathElements: FileLogPathElements,
	OutputModeEnvVar:    OutputModeEnvVar,
	RootErrorMessage:    ErrorMessagePreamble,
	LoggingContext:      []interface{}{"version", version.Version},
}
