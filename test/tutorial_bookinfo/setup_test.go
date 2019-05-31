package tutorial_bookinfo

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/solo-io/glooshot/pkg/cli"

	"github.com/solo-io/glooshot/test/utils"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/solo-io/glooshot/pkg/api/v1"
	"github.com/solo-io/glooshot/pkg/cli/gsutil"
	"github.com/solo-io/go-utils/testutils/goimpl"
	"github.com/solo-io/go-utils/testutils/kube"

	buildv1 "github.com/solo-io/build/pkg/api/v1"
	"github.com/solo-io/build/pkg/ingest"
	sgv1 "github.com/solo-io/supergloo/pkg/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/solo-io/solo-kit/pkg/api/v1/clients"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type clientSet struct {
	expClient  v1.ExperimentClient
	repClient  v1.ReportClient
	rrClient   sgv1.RoutingRuleClient
	meshClient sgv1.MeshClient
	kubeClient kubernetes.Interface
}

func setTestResources() {
	kubeClient := kube.MustKubeClient()
	ctx := context.Background()
	expClient, err := gsutil.GetExperimentClient(ctx, false)
	Expect(err).NotTo(HaveOccurred())
	repClient, err := gsutil.GetReportClient(ctx, false)
	Expect(err).NotTo(HaveOccurred())
	rrClient, err := gsutil.GetRoutingRuleClient(ctx, false)
	Expect(err).NotTo(HaveOccurred())
	meshClient, err := gsutil.GetMeshClient(ctx, false)
	Expect(err).NotTo(HaveOccurred())
	cs := clientSet{
		expClient:  expClient,
		repClient:  repClient,
		rrClient:   rrClient,
		meshClient: meshClient,
		kubeClient: kubeClient,
	}
	buildRun, err := ingest.InitializeBuildRun("../../solo-project.yaml", &buildv1.BuildEnvVars{})
	Expect(err).NotTo(HaveOccurred())
	gtr = testResources{
		cs:                cs,
		buildId:           buildRun.Config.BuildEnvVars.BuildId,
		GlooshotNamespace: "glooshot",
		IstioNamespace:    "istio-system",
		AppNamespace:      "bookinfo",

		// values used in tutorial
		tut: tutorialValues{
			meshName:         "istio-istio-system",
			rrVulnerableName: "reviews-vulnerable",
		},

		cleanupResources:      nil,
		portForwardAppCancel:  nil,
		portForwardPromCancel: nil,
		localGlooshotCancel:   nil,
	}
}

type testResources struct {
	buildId               string
	cs                    clientSet
	GlooshotNamespace     string
	IstioNamespace        string
	AppNamespace          string
	tut                   tutorialValues
	cleanupResources      []crd
	portForwardAppCancel  context.CancelFunc
	portForwardPromCancel context.CancelFunc
	localGlooshotCancel   context.CancelFunc
}
type tutorialValues struct {
	meshName         string
	rrVulnerableName string
}
type crd struct {
	resource  string
	namespace string
	name      string
}

func setupCluster() {
	setupGlooshotInit()
	setupIstio()
	setupLabelAppNamespace()
	setupPromStats()
	setupDeployBookinfo()
	setupRoutingRuleToVulnerableApp()
	setupLocalTestEnv()
	setupApplyFirstExperiment()
	setupProduceTraffic()
}

func restoreCluster() {
	cleanupResources()
	terminateAppPortForward()
	cleanupLocalTestEnv()
	cleanupMesh()
}

func cleanupResources() {
	for _, crd := range gtr.cleanupResources {
		cmdArgs := []string{"delete", crd.resource, "-n", crd.namespace, crd.name}
		cmd := exec.Command("kubectl", cmdArgs...)
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		err := cmd.Run()
		if err != nil {
			fmt.Printf("error with cmd %v, %v", cmdArgs, err)
		}
	}
}

func pushCleanup(c crd) {
	gtr.cleanupResources = append(gtr.cleanupResources, c)
}

func setupLocalTestEnv() {
	if os.Getenv("RUN_GLOOSHOT_LOCAL") != "1" {
		return
	}
	// port-forward prometheus
	Eventually(expectPortForwardPromRetry, 120*time.Second, 3*time.Second).Should(BeNil())
	// delete the glooshot deployment
	err := gtr.cs.kubeClient.AppsV1().Deployments(gtr.GlooshotNamespace).Delete("glooshot", &metav1.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())
	// run glooshot locally
	ctx, cancel := context.WithCancel(context.Background())
	gtr.localGlooshotCancel = cancel
	utils.RunGlooshotLocal(ctx, "http://localhost:9090")
}
func expectPortForwardPromRetry() error {
	portForwardProm()
	time.Sleep(3 * time.Second)
	if err := expectPortForwardPromReady(); err != nil {
		terminatePromPortForward()
		return err
	}
	return nil
}
func portForwardProm() {
	// we will be restarting the prom server when we update its config with supergloo
	// we need to find the new pod and connect to that
	promPod, err := getPromPod()
	if err != nil {
		return
	}
	promPodName := promPod.Name
	if err := promIsStable(); err != nil {
		return
	}
	localPort := 9090
	cmdString := fmt.Sprintf("port-forward -n %v %v %v:9090",
		gtr.GlooshotNamespace,
		promPodName,
		localPort)
	ctx, cancel := context.WithCancel(context.Background())
	gtr.portForwardPromCancel = cancel
	go kubectlWithCancel(ctx, cmdString)
}
func getPromPod() (corev1.Pod, error) {
	list, err := gtr.cs.kubeClient.CoreV1().Pods(gtr.GlooshotNamespace).List(metav1.ListOptions{LabelSelector: "app=prometheus,component=server"})
	if err != nil {
		return corev1.Pod{}, err
	}
	runningList := corev1.PodList{}
	for _, p := range list.Items {
		if p.Status.Phase == corev1.PodRunning {
			runningList.Items = append(runningList.Items, p)
		}
	}
	if len(runningList.Items) != 1 {
		return corev1.Pod{}, fmt.Errorf("no pods found")
	}
	promPod := list.Items[0]
	return promPod, nil
}
func promIsStable() error {
	promPod, err := getPromPod()
	if err != nil {
		return fmt.Errorf("could not get prometheus pod")
	}
	promReady := areAllContainersReady(promPod)
	if !promReady {
		return fmt.Errorf("prometheus is not ready")
	}
	return nil
}
func areAllContainersReady(pod corev1.Pod) bool {
	nContainersNotReady := 0
	for _, stat := range pod.Status.ContainerStatuses {
		if stat.State.Running == nil {
			nContainersNotReady++
		}
	}
	return nContainersNotReady == 0
}
func expectPortForwardPromReady() error {
	_, err := goimpl.Curl("http://localhost:9090")
	return err
}

func cleanupLocalTestEnv() {
	if os.Getenv("RUN_GLOOSHOT_LOCAL") != "1" {
		return
	}
	if gtr.localGlooshotCancel != nil {
		gtr.localGlooshotCancel()
		gtr.localGlooshotCancel = nil
	}
	terminatePromPortForward()
}
func terminatePromPortForward() {
	if gtr.portForwardPromCancel != nil {
		gtr.portForwardPromCancel()
		gtr.portForwardPromCancel = nil
	}
}

func cleanupMesh() {
	// in order to restore the prom config for the next test, we need to reset mesh.monitoringConfig.prometheusConfigmaps:
	// deleting the mesh and letting it get rediscovered by supergloo is an easy way to do that
	err := gtr.cs.meshClient.Delete(gtr.GlooshotNamespace, gtr.tut.meshName, clients.DeleteOpts{})
	if err != nil {
		fmt.Printf("error deleting mesh: %v\n", err)
	}
}

func setupGlooshotInit() {
	if ready(expectSetupGlooshotInitReady()) {
		fmt.Println("skipping glooshot init, already ready")
		return
	}
	out, err := cli.GlooshotConfig.RunForTest(fmt.Sprintf("init -f ../../_output/helm/charts/glooshot-%v.tgz", gtr.buildId))
	Expect(err).NotTo(HaveOccurred())
	fmt.Println(out.CobraStderr)
	fmt.Println(out.CobraStdout)
	Eventually(expectSetupGlooshotInitReady, 180*time.Second, 250*time.Millisecond).Should(BeNil())
}
func expectSetupGlooshotInitReady() error {
	list, err := gtr.cs.kubeClient.CoreV1().Pods(gtr.GlooshotNamespace).List(metav1.ListOptions{LabelSelector: "glooshot=glooshot-op"})
	if err != nil {
		return err
	}
	if len(list.Items) == 0 || len(list.Items) > 1 {
		return fmt.Errorf("no glooshot pods available")
	}
	return expectMatch(list.Items[0].Status.Phase, corev1.PodRunning)
}

func setupIstio() {
	if ready(expectSetupIstioReady()) {
		fmt.Println("skipping setup Istio, already ready")
		return
	}
	cmdString := "install istio --namespace glooshot --name istio-istio-system --installation-namespace istio-system --mtls=true --auto-inject=true"
	cmd := generateSuperglooCmd(cmdString)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	err := cmd.Run()
	//err := sgutils.Supergloo(cmdString)
	Expect(err).NotTo(HaveOccurred())
	Eventually(expectSetupIstioReady, 80*time.Second, 250*time.Millisecond).Should(BeNil())
}
func expectSetupIstioReady() error {
	list, err := gtr.cs.kubeClient.CoreV1().Pods(gtr.IstioNamespace).List(metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())
	if len(list.Items) == 0 {
		return fmt.Errorf("no istio pods found")
	}
	nPodsGettingReady := 0
	for _, p := range list.Items {
		if p.Status.Phase == corev1.PodPending {
			nPodsGettingReady++
		}
	}
	got := list.Items[0].Status.Phase
	expected := corev1.PodRunning
	return expectMatch(got, expected)
}
func expectMatch(got, expected interface{}) error {
	if got == expected {
		return nil
	}
	return fmt.Errorf("got: %v, expected: %v", got, expected)
}

func setupLabelAppNamespace() {
	if ready(expectSetupLabelAppNamespaceReady()) {
		fmt.Println("skipping setup label app namespace, already ready")
		return
	}
	_, _ = gtr.cs.kubeClient.CoreV1().Namespaces().Create(&corev1.Namespace{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: gtr.AppNamespace,
		},
	})
	cmdString := fmt.Sprintf("label namespace %v istio-injection=enabled", gtr.AppNamespace)
	cmd := exec.Command("kubectl", strings.Split(cmdString, " ")...)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	Eventually(expectSetupLabelAppNamespaceReady, 80*time.Second, 250*time.Millisecond).Should(BeNil())
}
func expectSetupLabelAppNamespaceReady() error {
	ns, err := gtr.cs.kubeClient.CoreV1().Namespaces().Get(gtr.AppNamespace, metav1.GetOptions{})
	if err != nil {
		return err
	}
	key := "istio-injection"
	val, ok := ns.ObjectMeta.Labels[key]
	if !ok {
		return fmt.Errorf("unable to read label key: %v", key)
	}
	expected := "enabled"
	if val == expected {
		return nil
	}
	return fmt.Errorf("got: %v, expected: %v", val, expected)
}

func generateSuperglooCmd(cmdString string) *exec.Cmd {
	cmd := exec.Command("supergloo", strings.Split(cmdString, " ")...)
	return cmd
}
func setupPromStats() {
	if ready(expectSetupPromStatsReady()) {
		fmt.Println("skipping setup prom stats, already ready")
		return
	}
	Eventually(getIstioMeshCrd, 180*time.Second, 1*time.Second).Should(BeNil())
	cmdString := "set mesh stats " +
		"--target-mesh glooshot.istio-istio-system " +
		"--prometheus-configmap glooshot.glooshot-prometheus-server"
	cmd := generateSuperglooCmd(cmdString)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	Eventually(expectSetupPromStatsReady, 30*time.Second, 500*time.Millisecond).Should(BeNil())
	Eventually(promIsStable, 60*time.Second, 500*time.Millisecond).Should(BeNil())
}
func expectSetupPromStatsReady() error {
	cm, err := gtr.cs.kubeClient.CoreV1().ConfigMaps(gtr.GlooshotNamespace).Get("glooshot-prometheus-server", metav1.GetOptions{})
	Expect(err).NotTo(HaveOccurred())
	hasIstioMetrics := false
	for _, v := range cm.Data {
		// this is one of the metrics that supergloo injects
		matchV, _ := regexp.MatchString("istio-istio-system-istio-policy", v)
		if matchV {
			hasIstioMetrics = true
		}
	}
	if !hasIstioMetrics {
		return fmt.Errorf("no istio metrics found")
	}
	return nil
}
func getIstioMeshCrd() error {
	_, err := gtr.cs.meshClient.Read(gtr.GlooshotNamespace, gtr.tut.meshName, clients.ReadOpts{})
	return err
}

func setupDeployBookinfo() {
	if ready(expectSetupDeployBookinfoReady()) {
		fmt.Println("skipping setup deploy bookinfo, already ready")
		return
	}
	cmd := exec.Command("kubectl", strings.Split(
		fmt.Sprintf("apply -n %v -f https://raw.githubusercontent.com/solo-io/glooshot/master/examples/bookinfo/bookinfo.yaml", gtr.AppNamespace),
		" ")...)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	Eventually(expectSetupDeployBookinfoReady, 80*time.Second, 250*time.Millisecond).Should(BeNil())
}
func expectSetupDeployBookinfoReady() error {
	list, err := gtr.cs.kubeClient.CoreV1().Pods(gtr.AppNamespace).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	if len(list.Items) == 0 {
		return fmt.Errorf("no bookinfo pods found")
	}
	nPodsGettingReady := 0
	for _, p := range list.Items {
		if p.Status.Phase == corev1.PodPending {
			nPodsGettingReady++
		}
	}
	got := list.Items[0].Status.Phase
	expected := corev1.PodRunning

	if got == expected {
		return nil
	}
	return fmt.Errorf("got: %v, expected: %v", got, expected)
}

func setupRoutingRuleToVulnerableApp() {
	if ready(expectSetupRoutingRuleToVulnerableAppReady()) {
		fmt.Println("skipping setup rr vulnerable, already ready")
		return
	}
	cmdString := "apply routingrule trafficshifting " +
		"--namespace glooshot " +
		"--name reviews-vulnerable " +
		fmt.Sprintf("--dest-upstreams glooshot.%v-reviews-9080 ", gtr.AppNamespace) +
		"--target-mesh glooshot.istio-istio-system " +
		fmt.Sprintf("--destination glooshot.%v-reviews-v4-9080:1", gtr.AppNamespace)
	cmd := generateSuperglooCmd(cmdString)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	pushCleanup(crd{"routingrule", "glooshot", "reviews-vulnerable"})
	Eventually(expectSetupRoutingRuleToVulnerableAppReady, 80*time.Second, 250*time.Millisecond).Should(BeNil())
}

func expectSetupRoutingRuleToVulnerableAppReady() error {
	_, err := gtr.cs.rrClient.Read(gtr.GlooshotNamespace, gtr.tut.rrVulnerableName, clients.ReadOpts{})
	return err
}

func ready(e error) bool {
	if e != nil {
		return false
	}
	return true
}
func setupApplyFirstExperiment() {
	if ready(expectSetupApplyFirstExperimentReady()) {
		fmt.Println("skipping setup apply first experiment, already ready")
		return
	}
	expPath := "../../examples/bookinfo/fault-abort-ratings.yaml"
	cmdString := fmt.Sprintf("apply -f %v", expPath)
	kubectl(cmdString)
	pushCleanup(crd{"experiment", gtr.AppNamespace, "abort-ratings-metric"})
	pushCleanup(crd{"routingrule", gtr.AppNamespace, "abort-ratings-metric-0"})

	timeLimit := 45 * time.Second
	Eventually(expectSetupApplyFirstExperimentReady, timeLimit, 250*time.Millisecond).Should(BeNil())
}
func expectSetupApplyFirstExperimentReady() error {
	if _, err := gtr.cs.expClient.Read(gtr.AppNamespace, "abort-ratings-metric", clients.ReadOpts{}); err != nil {
		return fmt.Errorf("could not read experiment")
	}
	if _, err := gtr.cs.rrClient.Read(gtr.AppNamespace, "abort-ratings-metric-0", clients.ReadOpts{}); err != nil {
		return fmt.Errorf("could not read routing rule")
	}
	return nil
}

// return the cancel function so that we can kill long-running commands like port-forward
func kubectlWithCancel(ctx context.Context, cmdString string) {
	defer GinkgoRecover()
	fmt.Printf("running command: kubectl %v\n", cmdString)
	cmd := exec.CommandContext(ctx, "kubectl", strings.Split(cmdString, " ")...)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	err := cmd.Run()
	if err != nil {
		fmt.Printf("error while running cmd: kubectl %v, err: %v\n", cmdString, err)
	}
}
func kubectl(cmdString string) {
	cmd := exec.Command("kubectl", strings.Split(cmdString, " ")...)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
}

func portForwardApp() {
	localPort := 9080
	cmdString := fmt.Sprintf("port-forward -n %v deploy/productpage-v1 %v:9080", gtr.AppNamespace, localPort)
	ctx, cancel := context.WithCancel(context.Background())
	gtr.portForwardAppCancel = cancel
	go kubectlWithCancel(ctx, cmdString)
}
func terminateAppPortForward() {
	if gtr.portForwardAppCancel != nil {
		gtr.portForwardAppCancel()
		gtr.portForwardAppCancel = nil
	}
}

func setupProduceTraffic() {
	portForwardApp()
	successCount := 0
	Eventually(promIsStable, 90*time.Second, 500*time.Millisecond).Should(BeNil())
	// give prom a moment to run
	time.Sleep(1 * time.Second)
	// use a go routine so that we generate traffic while checking for the failure condition
	go func() {
		defer GinkgoRecover()
		Eventually(getNValidResponses(&successCount, 50), 60*time.Second, 25*time.Millisecond).Should(BeNil())
	}()
	// wait for prom q's to get scraped
	Eventually(expectExpToHaveFailed(gtr.AppNamespace, "abort-ratings-metric"), 60*time.Second, 1*time.Second).Should(BeNil())
	Eventually(expectExpFailureReport(gtr.AppNamespace, "abort-ratings-metric"), 15*time.Second, 250*time.Millisecond).Should(BeNil())
}
func getNValidResponses(successCount *int, targetCount int) func() error {
	// use a closure so we can increment a success rate across retries
	return func() error {
		if _, err := goimpl.Curl("http://localhost:9080/productpage?u=normal"); err != nil {
			return err
		}
		*successCount = *successCount + 1
		if *successCount > targetCount {
			return nil
		}
		return fmt.Errorf("waiting for %v successes, have %v", targetCount, *successCount)
	}
}
func expectExpToHaveFailed(namespace, name string) func() error {
	return func() error {
		exp, err := gtr.cs.expClient.Read(namespace, name, clients.ReadOpts{})
		if err != nil {
			return err
		}
		return expectMatch(exp.Result.State, v1.ExperimentResult_Failed)
	}
}
func expectExpFailureReport(expNamespace, expName string) func() error {
	repNamespace := expNamespace
	repName := expName
	return func() error {
		rep, err := gtr.cs.repClient.Read(repNamespace, repName, clients.ReadOpts{})
		if err != nil {
			return err
		}
		if len(rep.FailureConditionHistory) != 1 {
			return fmt.Errorf("failure condition history length %v, not 1", len(rep.FailureConditionHistory))
		}
		if rep.FailureConditionHistory[0] == nil {
			By(fmt.Sprintf("failure condition history[0]: %v", rep.FailureConditionHistory[0]))
			return fmt.Errorf("nil entry for failure condition history")
		}
		return nil
	}
}
