package tutorial_bookinfo

import (
	"testing"

	"github.com/avast/retry-go"
	"github.com/solo-io/go-utils/testutils/clusterlock"
	"github.com/solo-io/go-utils/testutils/kube"

	"github.com/solo-io/go-utils/testutils"

	"github.com/solo-io/solo-kit/test/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var locker *clusterlock.TestClusterLocker

func TestTutorialBookinfo(t *testing.T) {

	helpers.RegisterPreFailHandler(
		func() {
			testutils.PrintTrimmedStack()
		})
	helpers.RegisterCommonFailHandlers()
	helpers.SetupLog()
	RunSpecs(t, "Bookinfo Tutorial Suite")
}

// these are setup before the suite and available test-wide
var gtr = testResources{}

var _ = BeforeSuite(func() {
	var err error
	locker, err = clusterlock.NewTestClusterLocker(kube.MustKubeClient(), clusterlock.Options{})
	Expect(err).NotTo(HaveOccurred())
	Expect(locker.AcquireLock(retry.Attempts(20))).NotTo(HaveOccurred())
	setTestResources()
	setupCluster()
})

var _ = AfterSuite(func() {
	defer locker.ReleaseLock()
	restoreCluster()
})
