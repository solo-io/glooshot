package translator_test

import (
	"testing"

	"github.com/solo-io/go-utils/testutils"

	"github.com/solo-io/solo-kit/test/helpers"

	. "github.com/onsi/ginkgo"
)

func TestTranslator(t *testing.T) {

	helpers.RegisterPreFailHandler(
		func() {
			testutils.PrintTrimmedStack()
		})
	helpers.RegisterCommonFailHandlers()
	helpers.SetupLog()
	RunSpecs(t, "Translator Suite")
}
