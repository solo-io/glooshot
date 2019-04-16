package install

import (
	"testing"

	"github.com/solo-io/solo-kit/test/helpers"

	. "github.com/onsi/ginkgo"
)

func TestMakefile(t *testing.T) {

	helpers.RegisterCommonFailHandlers()
	helpers.SetupLog()
	RunSpecs(t, "Makefile Suite")
}
