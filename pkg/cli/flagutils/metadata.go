package flagutils

import (
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	"github.com/spf13/pflag"
)

func AddMetadataFlags(set *pflag.FlagSet, in *core.Metadata) {
	set.StringVar(&in.Name, "name", "", "name for the resource")
	set.StringVar(&in.Namespace, "namespace", "supergloo-system", "namespace for the resource")
}
