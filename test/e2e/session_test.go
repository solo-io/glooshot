package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/solo-io/go-utils/testutils/kube"
	"k8s.io/client-go/kubernetes"

	"github.com/solo-io/glooshot/pkg/gsutil"
	"github.com/solo-io/glooshot/test/data"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/setup"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Glooshot", func() {

	var (
		ctx       context.Context
		client    v1.ExperimentClient
		clientset kubernetes.Interface
		namespace string
		name1     = "testexperiment1"
		name2     = "testexperiment2"
		name3     = "testexperiment3"
		//url       = "http:http//localhost:8085"
		err error
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = randomNamespace("glooshot-test")
		client, err = gsutil.GetExperimentClient(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		go func() {
			err := setup.Run(ctx)
			Expect(err).NotTo(HaveOccurred())
		}()
		clientset = kube.MustKubeClient()
		_, err := clientset.Core().Namespaces().Create(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		var zero int64
		zero = 0
		clientset.Core().Namespaces().Delete(namespace, &metav1.DeleteOptions{GracePeriodSeconds: &zero})
	})

	It("should watch for experiment crds", func() {
		exp1 := getNewExperiment(namespace, name1)
		_, err = client.Write(exp1, clients.WriteOpts{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() int {
			exps, err := client.List(namespace, clients.ListOpts{})
			Expect(err).NotTo(HaveOccurred())
			return len(exps)
		}).Should(BeNumerically("==", 1))

		exp2 := data.GetBasicExperimentAbort(namespace, name2)
		_, err = client.Write(exp2, clients.WriteOpts{})
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() int {
			exps, err := client.List(namespace, clients.ListOpts{})
			Expect(err).NotTo(HaveOccurred())
			return len(exps)
		}).Should(BeNumerically("==", 2))

		exp3 := data.GetBasicExperimentDelay(namespace, name3)
		_, err = client.Write(exp3, clients.WriteOpts{})
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() int {
			exps, err := client.List(namespace, clients.ListOpts{})
			Expect(err).NotTo(HaveOccurred())
			return len(exps)
		}).Should(BeNumerically("==", 3))

	})
})

func getNewExperiment(namespace, name string) *v1.Experiment {
	return &v1.Experiment{
		Metadata: core.Metadata{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// TODO(mitchdraft) migrate this to go-utils https://github.com/solo-io/glooshot/issues/16
func curl(url string) (string, error) {
	body := bytes.NewReader([]byte(url))
	req, err := http.NewRequest("GET", url, body)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	p := new(bytes.Buffer)
	_, err = io.Copy(p, resp.Body)
	defer resp.Body.Close()

	return p.String(), nil
}

// from openshift
func randomNamespace(prefix string) string {
	return prefix + string([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))[3:12])
}
