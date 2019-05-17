package tutorial_bookinfo

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/solo-io/glooshot/pkg/setup/options"

	"github.com/solo-io/glooshot/pkg/translator"

	"github.com/solo-io/go-utils/testutils/kube"
	"k8s.io/client-go/kubernetes"

	"github.com/solo-io/glooshot/pkg/cli/gsutil"
	"github.com/solo-io/glooshot/test/data"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/setup"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	sgv1 "github.com/solo-io/supergloo/pkg/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Glooshot", func() {

	var (
		ctx       context.Context
		cs        clientSet
		namespace string
		name1     = "testexperiment1"
		name2     = "testexperiment2"
		name3     = "testexperiment3"
	)

	BeforeEach(func() {
		namespace = randomNamespace("glooshot-test")
		kubeClient := kube.MustKubeClient()
		_, err := kubeClient.CoreV1().Namespaces().Create(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})
		Expect(err).NotTo(HaveOccurred())
		ctx = context.Background()
		expClient, err := gsutil.GetExperimentClient(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		repClient, err := gsutil.GetReportClient(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		rrClient, err := gsutil.GetRoutingRuleClient(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		meshClient, err := gsutil.GetMeshClient(ctx, false)
		Expect(err).NotTo(HaveOccurred())
		cs = clientSet{
			expClient:  expClient,
			repClient:  repClient,
			rrClient:   rrClient,
			meshClient: meshClient,
			kubeClient: kubeClient,
		}
		// TODO(mitchdraft) - restore when prometheus is deployed to the test env.
		if os.Getenv("CI_TESTS") == "1" {
			fmt.Printf("this test is disabled in CI. to run, ensure env var `CI_TESTS` is not set to 1")
			return
		}
		go func() {
			defer GinkgoRecover()
			testOpts := options.DefaultOpts()
			testOpts.PrometheusURL = "http://localhost:9090"
			err := setup.Run(ctx, testOpts)
			By(fmt.Sprintf("goroutine running with error: %v", err))
			Expect(err).NotTo(HaveOccurred())
		}()
	})

	AfterEach(func() {
		var zero int64
		zero = 0
		_ = cs.kubeClient.CoreV1().Namespaces().Delete(namespace, &metav1.DeleteOptions{GracePeriodSeconds: &zero})
	})

	It("should watch for experiment crds", func() {
		// TODO(mitchdraft) - restore when prometheus is deployed to the test env.
		if os.Getenv("CI_TESTS") == "1" {
			fmt.Printf("this test is disabled in CI. to run, ensure env var `CI_TESTS` is not set to 1")
			return
		}
		exp1 := getNewExperiment(namespace, name1)
		cs.createAndWait(exp1, 1, 0)

		exp2 := data.GetBasicExperimentAbort(namespace, name2)
		cs.createAndWait(exp2, 2, 1)

		exp3 := data.GetBasicExperimentDelay(namespace, name3)
		cs.createAndWait(exp3, 3, 1)
	})

})

type clientSet struct {
	expClient  v1.ExperimentClient
	repClient  v1.ReportClient
	rrClient   sgv1.RoutingRuleClient
	meshClient sgv1.MeshClient
	kubeClient kubernetes.Interface
}

func (cs clientSet) createAndWait(exp *v1.Experiment, expCount, rrCount int) {
	_, err := cs.expClient.Write(exp, clients.WriteOpts{})
	Expect(err).NotTo(HaveOccurred())
	Eventually(func() int {
		exps, err := cs.expClient.List(exp.Metadata.Namespace, clients.ListOpts{})
		Expect(err).NotTo(HaveOccurred())
		return len(exps)
	}).Should(BeNumerically("==", expCount))
	Eventually(func() int {
		//rrs, err := cs.rrClient.List(exp.Metadata.Namespace, clients.ListOpts{Selector: map[string]string{translator.RoutingRuleLabelKey: exp.Metadata.Name}})
		rrs, err := cs.rrClient.List(exp.Metadata.Namespace, clients.ListOpts{Selector: translator.LabelsForRoutingRule(exp.Metadata.Name)})
		Expect(err).NotTo(HaveOccurred())
		return len(rrs)
	}, 3*time.Second, 250*time.Millisecond).Should(BeNumerically("==", rrCount))
}

func getNewExperiment(namespace, name string) *v1.Experiment {
	return &v1.Experiment{
		Metadata: core.Metadata{
			Namespace: namespace,
			Name:      name,
		},
	}
}

// from openshift
func randomNamespace(prefix string) string {
	return prefix + string([]byte(fmt.Sprintf("%d", time.Now().UnixNano()))[3:12])
}

func createMesh(meshClient sgv1.MeshClient, namespace, name string) {
	mesh := &sgv1.Mesh{
		Metadata: core.Metadata{
			Name:      name,
			Namespace: namespace,
		},
		MeshType: nil,
	}
	_, err := meshClient.Write(mesh, clients.WriteOpts{})
	Expect(err).NotTo(HaveOccurred())
}
