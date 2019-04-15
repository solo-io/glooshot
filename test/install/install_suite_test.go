package install

import (
	"github.com/solo-io/go-utils/logger"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/testutils/exec"

	"github.com/solo-io/solo-kit/test/helpers"

	. "github.com/onsi/ginkgo"
)

func TestInstall(t *testing.T) {

	envToggleKey := "RUN_GLOOSHOT_INSTALL_TESTS"
	envToggleValue := "1"
	if os.Getenv(envToggleKey) != envToggleValue {
		logger.Warnf("This test requires a running kubernetes cluster and built images. It is disabled by default. "+
			"To enable, set %s=%s in your env.", envToggleKey, envToggleValue)
		return
	}

	helpers.RegisterCommonFailHandlers()
	helpers.SetupLog()
	RunSpecs(t, "Install Suite")
}

const glooshotManifest = "../../install/glooshot.yaml"

var _ = BeforeSuite(func() {
	// install glooshot via the manifest file
	err := toggleManifest(glooshotManifest, true)
	Expect(err).NotTo(HaveOccurred())
})
var _ = AfterSuite(func() {
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
