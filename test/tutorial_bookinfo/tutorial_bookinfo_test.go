package tutorial_bookinfo

import (
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Glooshot", func() {

	It("should complete the bookinfo tutorial successfully", func() {
		// TODO(mitchdraft) - restore when prometheus is deployed to the test env.
		if os.Getenv("CI_TESTS") == "1" {
			fmt.Printf("this test is disabled in CI. to run, ensure env var `CI_TESTS` is not set to 1")
			return
		}
	})

})
