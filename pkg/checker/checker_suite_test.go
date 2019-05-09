package checker_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestChecker(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Checker Suite")
}
