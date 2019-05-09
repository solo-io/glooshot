package install

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/solo-io/go-utils/testutils/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = XDescribe("Glooshot", func() {

	// TODO - make this configurable through the helm chart
	var glooshotNamespace = "glooshot"

	It("should contain glooshot", func() {
		kc := kube.MustKubeClient()
		Eventually(func() (int, error) {
			pods, err := kc.CoreV1().Pods(glooshotNamespace).List(metav1.ListOptions{})
			return len(pods.Items), err
		}, "5s", "0.5s").Should(BeNumerically(">", 0))
		Eventually(func() (string, error) {
			pods, err := kc.CoreV1().Pods(glooshotNamespace).List(metav1.ListOptions{})
			return pods.Items[0].Spec.Containers[0].Image, err
		}, "5s", "0.5s").Should(MatchRegexp("glooshot-op"))
	})
})
