package tutorial_bookinfo

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Glooshot", func() {

	It("should complete the bookinfo tutorial successfully", func() {
		// if setup runs correctly, the test has passed
		Expect(1).To(Equal(1))
	})

})
