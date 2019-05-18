package utils

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/glooshot/pkg/setup"
	"github.com/solo-io/glooshot/pkg/setup/options"
)

func RunGlooshotLocal(ctx context.Context, promUrl string) {
	go func() {
		defer GinkgoRecover()
		testOpts := options.DefaultOpts()
		testOpts.PrometheusURL = promUrl
		err := setup.Run(ctx, testOpts)
		By(fmt.Sprintf("goroutine running with error: %v", err))
		Expect(err).NotTo(HaveOccurred())
	}()
}
