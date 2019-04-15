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
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		time.Sleep(4 * time.Second)
		ExpectWithOffset(1, len(pods.Items)).To(BeNumerically(">", 0))
		ExpectWithOffset(1, pods.Items[0].Spec.Containers[0].Image).To(MatchRegexp("glooshot-op"))
	})
})
