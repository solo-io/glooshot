package initexp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/github"
	"github.com/solo-io/glooshot/pkg/version"
	"golang.org/x/oauth2"

	"github.com/solo-io/glooshot/pkg/cli/options"
	"github.com/spf13/cobra"

	"github.com/pkg/errors"
	"github.com/solo-io/go-utils/installutils/helmchart"
	v1 "k8s.io/api/core/v1"
	kubeerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	sgChartUriTemplate  = "https://storage.googleapis.com/glooshot-helm/charts/glooshot-%s.tgz"
	tmpInstallNamespace = "glooshot"
)

func Cmd(opts *options.Options) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "install SuperGloo to a Kubernetes cluster",
		Long: `Installs SuperGloo using default values based on the official helm chart located in install/helm/glooshot

The basic SuperGloo installation is composed of single-instance deployments for the glooshot-controller and discovery pods. 
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := installGlooshot(opts); err != nil {
				return errors.Wrapf(err, "installing glooshot")
			}
			return nil
		},
	}
	//flagutils.AddInitFlags(cmd.PersistentFlags(), &opts.Init)
	return cmd
}

func installGlooshot(opts *options.Options) error {
	releaseVersion, err := getReleaseVersion(opts)
	if err != nil {
		return errors.Wrapf(err, "getting release version")
	}

	// Get location of Gloo helm chart
	chartUri := fmt.Sprintf(sgChartUriTemplate, releaseVersion)
	if helmChartOverride := opts.Init.HelmChartOverride; helmChartOverride != "" {
		chartUri = helmChartOverride
	}

	values, err := readValues(opts.Init.HelmValues)
	if err != nil {
		return errors.Wrapf(err, "reading custom values")
	}

	if _, err := opts.Clients.KubeClient().CoreV1().Namespaces().Create(&v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: tmpInstallNamespace},
	}); err != nil && !kubeerrs.IsAlreadyExists(err) {
		return errors.Wrapf(err, "creating namespace")
	}

	manifests, err := helmchart.RenderManifests(opts.Ctx,
		chartUri,
		values,
		"glooshot",
		tmpInstallNamespace,
		"",
	)
	if err != nil {
		return errors.Wrapf(err, "rendering manifests")
	}

	manifest := manifests.CombinedString()

	if opts.Init.DryRun {
		fmt.Printf("%s\n", manifest)
		return nil
	}

	fmt.Printf("installing glooshot version %v\nusing chart uri %v\n", releaseVersion, chartUri)

	if err := kubectlApply(manifest); err != nil {
		return errors.Wrapf(err, "executing kubectl failed")
	}

	fmt.Printf("install successful!\n")
	return nil
}

func getReleaseVersion(opts *options.Options) (string, error) {
	//if !version.IsReleaseVersion() {
	//	if opts.Init.ReleaseVersion == "" {
	//		return "", errors.Errorf("you must provide a " +
	//			"release version containing the manifest when " +
	//			"running an unreleased version of glooshot.")
	//	} else if opts.Init.ReleaseVersion == "latest" {
	//		releaseVersion, err := helpers.GetLatestVersion(opts.Ctx, "glooshot")
	//		if err != nil {
	//			return "", errors.Wrapf(err, "unable to retrieve latest release version from github")
	//		}
	//		return releaseVersion, nil
	//	}
	//	return opts.Init.ReleaseVersion, nil
	//}
	return version.Version, nil
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

func kubectlApply(manifest string) error {
	return kubectl(bytes.NewBufferString(manifest), "apply", "-f", "-")
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

func GetLatestVersion(ctx context.Context, repo string) (string, error) {
	client := GetPublicGithubClient(ctx)
	release, _, err := client.Repositories.GetLatestRelease(ctx, "solo-io", repo)
	if err != nil {
		return "", errors.Wrapf(err, "unable to get latest version for %s", repo)
	}
	return release.GetTagName()[1:], nil
}

func IsValidVersion(ctx context.Context, repo, version string) (string, error) {
	version = "v" + strings.TrimPrefix(version, "v")
	client := GetPublicGithubClient(ctx)
	release, _, err := client.Repositories.GetReleaseByTag(ctx, "solo-io", repo, version)
	if err != nil {
		return "", errors.Wrapf(err, "%s is not a valid version for %s", version, repo)
	}
	return release.GetTagName()[1:], nil
}

func GetPublicGithubClient(ctx context.Context) *github.Client {
	client := http.DefaultClient
	if githubToken := os.Getenv("GITHUB_TOKEN"); githubToken != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: githubToken},
		)
		client = oauth2.NewClient(ctx, ts)
	}
	return github.NewClient(client)
}
