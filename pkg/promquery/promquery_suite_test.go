package promquery_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestPromquery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Promquery Suite")
}
