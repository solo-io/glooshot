package install

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/testutils/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Glooshot", func() {

	// TODO - make this configurable through the helm chart
	var glooshotNamespace = "glooshot"

	It("should contain glooshot", func() {
		kc := kube.MustKubeClient()
		pods, err := kc.CoreV1().Pods(glooshotNamespace).List(metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(len(pods.Items)).To(BeNumerically(">", 0))
		time.Sleep(100 * time.Second)
		Expect(pods.Items[0].Spec.Containers[0].Image).To(Equal("g"))
	})
})
