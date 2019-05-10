package starter_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStarter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Starter Suite")
}
