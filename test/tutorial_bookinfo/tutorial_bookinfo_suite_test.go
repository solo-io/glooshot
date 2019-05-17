package tutorial_bookinfo

import (
	"fmt"
	"os"
	"testing"

	"github.com/solo-io/go-utils/testutils"

	"github.com/solo-io/solo-kit/test/helpers"

	. "github.com/onsi/ginkgo"
)

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
	// set up the cluster
	if os.Getenv("CI_TESTS") == "1" {
		fmt.Printf("this test is disabled in CI. to run, ensure env var `CI_TESTS` is not set to 1")
		return
	}

	setTestResources()
	setupCluster()
})
var _ = AfterSuite(func() {
	// set up the cluster
	if os.Getenv("CI_TESTS") == "1" {
		fmt.Printf("this test is disabled in CI. to run, ensure env var `CI_TESTS` is not set to 1")
		return
	}

	restoreCluster()
})
