package e2e_test

import (
	"testing"

	"github.com/solo-io/solo-kit/test/helpers"

	. "github.com/onsi/ginkgo"
)

func TestE2e(t *testing.T) {

	helpers.RegisterCommonFailHandlers()
	helpers.SetupLog()
	RunSpecs(t, "E2e Suite")
}
