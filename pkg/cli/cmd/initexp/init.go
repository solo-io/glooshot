package initexp

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/solo-io/glooshot/pkg/version"

	"github.com/solo-io/glooshot/pkg/cli/flagutils"

	"github.com/solo-io/glooshot/pkg/cli/options"
	"github.com/spf13/cobra"

	"github.com/pkg/errors"
	"github.com/solo-io/go-utils/installutils/helmchart"
	v1 "k8s.io/api/core/v1"
	kubeerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	gsChartUriTemplate = "https://storage.googleapis.com/glooshot-helm/charts/glooshot-%s.tgz"
)

func Cmd(opts *options.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "install Glooshot to a Kubernetes cluster",
		Long:  `Installs Glooshot using default values based on the official helm chart located in install/helm/glooshot`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := installGlooshot(opts); err != nil {
				return errors.Wrapf(err, "installing glooshot")
			}
			return nil
		},
	}
	flagutils.AddInitFlags(cmd.PersistentFlags(), &opts.Init)
	return cmd
}

func installGlooshot(opts *options.Options) error {
	releaseVersion := version.Version

	// Get location of Gloo helm chart
	chartUri := fmt.Sprintf(gsChartUriTemplate, releaseVersion)
	if helmChartOverride := opts.Init.HelmChartOverride; helmChartOverride != "" {
		chartUri = helmChartOverride
	}

	values, err := readValues(opts.Init.HelmValues)
	if err != nil {
		return errors.Wrapf(err, "reading custom values")
	}

	if _, err := opts.Clients.KubeClient().CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: opts.Init.InstallNamespace},
	}); err != nil && !kubeerrs.IsAlreadyExists(err) {
		return errors.Wrapf(err, "creating namespace")
	}

	manifests, err := helmchart.RenderManifests(opts.Ctx,
		chartUri,
		values,
		"glooshot",
		opts.Init.InstallNamespace,
		"",
	)
	if err != nil {
		return errors.Wrapf(err, "rendering manifest from uri: %v", chartUri)
	}

	manifest := manifests.CombinedString()

	if opts.Init.DryRun {
		fmt.Printf("%s\n", manifest)
		return nil
	}

	fmt.Printf("installing glooshot version %v\nusing chart uri %v\n", releaseVersion, chartUri)

	if err := kubectlApply(manifest, opts.Init.InstallNamespace); err != nil {
		return errors.Wrapf(err, "executing kubectl failed")
	}

	fmt.Printf("install successful!\n")
	return nil
}

func readValues(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func kubectlApply(manifest, namespace string) error {
	return kubectl(bytes.NewBufferString(manifest), "apply", "-n", namespace, "-f", "-")
}

func kubectl(stdin io.Reader, args ...string) error {
	kubectl := exec.Command("kubectl", args...)
	if stdin != nil {
		kubectl.Stdin = stdin
	}
	kubectl.Stdout = os.Stdout
	kubectl.Stderr = os.Stderr
	return kubectl.Run()
}
