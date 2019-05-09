package install

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/solo-io/go-utils/contextutils"
	"github.com/solo-io/solo-kit/test/helpers"

	"github.com/avast/retry-go"
	"github.com/solo-io/go-utils/testutils"
	"github.com/solo-io/go-utils/testutils/clusterlock"
	"github.com/solo-io/go-utils/testutils/kube"

	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/testutils/exec"

	. "github.com/onsi/ginkgo"
)

var locker *clusterlock.TestClusterLocker

func TestInstall(t *testing.T) {
	envToggleKey := "RUN_GLOOSHOT_INSTALL_TESTS"
	envToggleValue := "1"
	if os.Getenv(envToggleKey) != envToggleValue {
		contextutils.LoggerFrom(context.TODO()).Warnf("This test requires a running kubernetes cluster and built images. It is disabled by default. "+
			"To enable, set %s=%s in your env.", envToggleKey, envToggleValue)
		return
	}
	helpers.RegisterPreFailHandler(
		func() {
			testutils.PrintTrimmedStack()
		})
	helpers.RegisterCommonFailHandlers()
	helpers.SetupLog()
	RunSpecs(t, "Install Suite")
}

const glooshotManifest = "../../install/glooshot.yaml"

var _ = BeforeSuite(func() {
	var err error
	locker, err = clusterlock.NewTestClusterLocker(kube.MustKubeClient(), clusterlock.Options{})
	Expect(err).NotTo(HaveOccurred())
	Expect(locker.AcquireLock(retry.Attempts(20))).NotTo(HaveOccurred())

	err = printManifestSummary(glooshotManifest, false, nil, true)
	Expect(err).NotTo(HaveOccurred())
	// install glooshot via the manifest file
	err = toggleManifest(glooshotManifest, true)
	Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
	defer locker.ReleaseLock()
	// uninstall glooshot via the manifest file
	err := toggleManifest(glooshotManifest, false)
	Expect(err).NotTo(HaveOccurred())
})

// apply or delete a manifest with kubectl
func toggleManifest(manifestFilepath string, enable bool) error {
	kubectlSpec := []string{"kubectl"}
	if enable {
		kubectlSpec = append(kubectlSpec, "apply")
	} else {
		kubectlSpec = append(kubectlSpec, "delete")
	}
	kubectlSpec = append(kubectlSpec, []string{"-f", manifestFilepath}...)
	return exec.RunCommand(".", true, kubectlSpec...)
}

func printManifestSummary(file string, entire bool, matchers []string, useDefaults bool) error {
	if entire {
		manifest, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		fmt.Printf("glooshot.yaml manifest:\n%s", string(manifest))
		return nil
	}
	defaults := []string{"image"}
	if useDefaults {
		matchers = append(matchers, defaults...)
	}
	return printFileMatchers(file, matchers)
}

func printFileMatchers(file string, matchers []string) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		for _, m := range matchers {
			if matched, _ := regexp.Match(m, sc.Bytes()); matched {
				fmt.Println(sc.Text())
				continue
			}
		}

	}
	return nil
}
