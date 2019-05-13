package flagutils

import (
	"github.com/solo-io/glooshot/pkg/cli/options"
	"github.com/solo-io/glooshot/pkg/version"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	"github.com/spf13/pflag"
)

func AddMetadataFlags(set *pflag.FlagSet, in *core.Metadata) {
	set.StringVar(&in.Name, "name", "", "name for the resource")
	set.StringVar(&in.Namespace, "namespace", "glooshot", "namespace for the resource")
}

func AddInitFlags(set *pflag.FlagSet, init *options.Init) {
	set.StringVarP(&init.HelmChartOverride, "file", "f", "", "Install Glooshot from this Helm chart rather than from a release. Target file must be a tarball")
	set.StringVarP(&init.HelmValues, "values", "v", "", "Provide a custom values.yaml overrides for the installed helm chart. Leave empty to use default values from the chart.")
	set.StringVarP(&init.InstallNamespace, "namespace", "n", "glooshot", "namespace to install glooshot into")
	if !version.IsReleaseVersion() {
		set.StringVar(&init.ReleaseVersion, "release", "", "install from this release version. Should correspond with the "+
			"name of the release on GitHub")
	}
	set.BoolVarP(&init.DryRun, "dry-run", "d", false, "Dump the raw installation yaml instead of applying it to kubernetes")
}
