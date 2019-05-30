package tutorial_bookinfo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
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
	//setupUseMinikube()
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

const readyString = "ready: the eventually-function has returned as expected"

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
	Eventually(portForwardPromRetry, 120*time.Second, 3*time.Second).Should(BeTrue())
	// delete the glooshot deployment
	err := gtr.cs.kubeClient.AppsV1().Deployments(gtr.GlooshotNamespace).Delete("glooshot", &metav1.DeleteOptions{})
	Expect(err).NotTo(HaveOccurred())
	// run glooshot locally
	ctx, cancel := context.WithCancel(context.Background())
	gtr.localGlooshotCancel = cancel
	utils.RunGlooshotLocal(ctx, "http://localhost:9090")
}
func portForwardPromRetry() bool {
	portForwardProm()
	time.Sleep(3 * time.Second)
	if isPortForwardPromReady() {
		return true
	}
	terminatePromPortForward()
	return false
}
func portForwardProm() {
	// we will be restarting the prom server when we update its config with supergloo
	// we need to find the new pod and connect to that
	promPod, err := getPromPod()
	if err != nil {
		return
	}
	promPodName := promPod.Name
	stable := promIsStable()
	if !stable {
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
func promIsStable() bool {
	promPod, err := getPromPod()
	if err != nil {
		fmt.Println("could not get prom pod")
		return false
	}
	promReady := areAllContainersReady(promPod)
	if !promReady {
		return false
	}
	return true
}
func areAllContainersReady(pod corev1.Pod) bool {
	nContainersNotReady := 0
	for _, stat := range pod.Status.ContainerStatuses {
		if stat.State.Running == nil {
			nContainersNotReady++
			fmt.Println(stat.State)
		}
	}
	fmt.Println(nContainersNotReady)
	return nContainersNotReady == 0
}
func isPortForwardPromReady() bool {
	_, err := curl("http://localhost:9090")
	if err != nil {
		return false
	}
	return true
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

//func setupUseMinikube() {
//	buff := bytes.NewBuffer([]byte{})
//	cmd := exec.Command("minikube", "docker-env")
//	cmd.Stdout = buff
//	err := cmd.Run()
//	Expect(err).NotTo(HaveOccurred())
//	fmt.Println(buff.String())
//	//lines := strings.Split(buff.String(), "\n")
//	reTls := regexp.MustCompile("export DOCKER_TLS_VERIFY=\"(.*)\"")
//	reHost := regexp.MustCompile("export DOCKER_HOST=\"(.*)\"")
//	reCertPath := regexp.MustCompile("export DOCKER_CERT_PATH=\"(.*)\"")
//	reApiVersion := regexp.MustCompile("export DOCKER_API_VERSION=\"(.*)\"")
//	maTls := reTls.FindAllStringSubmatch(buff.String(), -1)
//	maHost := reHost.FindAllStringSubmatch(buff.String(), -1)
//	maCertPath := reCertPath.FindAllStringSubmatch(buff.String(), -1)
//	maApiVersion := reApiVersion.FindAllStringSubmatch(buff.String(), -1)
//	//for _, l := range lines {
//	//	matches := re.FindAllString(l, -1)
//	//	fmt.Println(matches)
//	//	matches2 := re.FindAllStringSubmatch(l, -1)
//	//	fmt.Println(matches2)
//	//	//regexp.MatchString()
//	//}
//	//export DOCKER_TLS_VERIFY="1"
//	//export DOCKER_HOST="tcp://192.168.99.100:2376"
//	//export DOCKER_CERT_PATH="/Users/mitch/.minikube/certs"
//	//export DOCKER_API_VERSION="1.35"
//}

// minikube docker-env
// export DOCKER_TLS_VERIFY="1"
// export DOCKER_HOST="tcp://192.168.99.100:2376"
// export DOCKER_CERT_PATH="/Users/mitch/.minikube/certs"
// export DOCKER_API_VERSION="1.35"
// # Run this command to configure your shell:
// # eval $(minikube docker-env)

//func applyDockerEnv(subName, body string) error {
//
//}

var vvv = `
---
title: Tutorial
menuTitle: Tutorial
weight: 3
---

## Bookinfo Tutorial

This tutorial will show you how to use Gloo Shot to apply chaos experiments to a simple service mesh app.
We will use a slight modification of the familiar bookinfo app from Istio's
[sample app repo](https://github.com/istio/istio/tree/master/samples/bookinfo). We have modified the reviews service to
include a vulnerability that can lead to cascading failure. We will use Gloo Shot to detect this weakness.


#### The Goal

Services should be built to be resilient when dependencies are unavailable in order to avoid cascading failures.
In this example, we show how to detect cascading failures: failures
where an error in one service disables other services that interact with it. In the diagram below, we show two versions
of a reviews service. The version on the top right fails when it does not receive a valid response from the ratings.
The version on the bottom right handles the error more gracefully. It still provides review information even though the
ratings data is not available.


{{< figure src="/tutorial/bookinfo_resilience_demo.png" title="The book info app consists of three services. If the ratings service fails, we do not want it to break the reviews service, as shown in the top-right frame. In a resilient app, the reviews service will continue to work, even if one of its dependencies is unavailable, as shown in the bottom-right frame." >}}

### Prerequisites

To follow this demo, you will need the following:

- <code>glooshot</code> [(download)](https://github.com/solo-io/glooshot/releases) command line tool, v0.0.4 or greater
- <code>supergloo</code> [(download)](https://supergloo.solo.io/installation/) command line tool, v0.3.18 or greater, for simplified mesh management during the tutorial.
- <code>kubectl</code> [(download)](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- A Kubernetes cluster - [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/#install-minikube) will do

### Setup

#### Deploy Gloo Shot

- Gloo Shot can easily be deployed from the command line tool.
- This will put Gloo Shot in the <code>glooshot</code> namespace.

<code>glooshot init</code>
`

func setupGlooshotInit() {
	if isSetupGlooshotInitReady() {
		fmt.Println("skipping glooshot init, already ready")
		return
	}
	out, err := cli.GlooshotConfig.RunForTest(fmt.Sprintf("init -f ../../_output/helm/charts/glooshot-%v.tgz", gtr.buildId))
	Expect(err).NotTo(HaveOccurred())
	fmt.Println(out.CobraStderr)
	fmt.Println(out.CobraStdout)
	Eventually(isSetupGlooshotInitReady, 180*time.Second, 250*time.Millisecond).Should(BeTrue())
}
func isSetupGlooshotInitReady() bool {
	list, err := gtr.cs.kubeClient.CoreV1().Pods(gtr.GlooshotNamespace).List(metav1.ListOptions{LabelSelector: "glooshot=glooshot-op"})
	Expect(err).NotTo(HaveOccurred())
	if len(list.Items) == 0 || len(list.Items) > 1 {
		return false
	}
	return list.Items[0].Status.Phase == corev1.PodRunning
}

/*

- Let's review what this command is doing:

```bash
kubectl get pods -n glooshot -w
```

- When the initialization is completed, you should see something like this:

```bash
kubectl get deployments -n glooshot
NAME                                     READY   UP-TO-DATE   AVAILABLE   AGE
discovery                                1/1     1            1           2m37s
glooshot                                 1/1     1            1           2m36s
glooshot-prometheus-alertmanager         1/1     1            1           2m37s
glooshot-prometheus-kube-state-metrics   1/1     1            1           2m37s
glooshot-prometheus-pushgateway          1/1     1            1           2m37s
glooshot-prometheus-server               1/1     1            1           2m37s
mesh-discovery                           1/1     1            1           2m36s
supergloo                                1/1     1            1           2m37s
```

- These resources serve the following purposes:
- **glooshot** manages your chaos experiments
- **supergloo** and **mesh-discovery** are from the [SuperGloo](https://supergloo.solo.io/). Together, they translate experiment specifications into the desired service mesh behavior.
- **discovery**, from [Gloo](https://supergloo.solo.io/), finds and lists all the available chaos experiment targets.
- **glooshot-prometheus-***, from [Prometheus](https://prometheus.io/), provides metrics. If you already have Prometheus running it is possible configure Gloo Shot to use your existing instance instead of deploying this one.

#### Install a service mesh (if you have not already)

- Install a service mesh.
- We will use Istio for this tutorial.
- We will use SuperGloo to install Istio with Prometheus.

```bash
supergloo install istio \
    --namespace glooshot \
    --name istio-istio-system \
    --installation-namespace istio-system \
    --mtls=true \
    --auto-inject=true
```
*/

func setupIstio() {
	if isSetupIstioReady() {
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
	Eventually(isSetupIstioReady, 80*time.Second, 250*time.Millisecond).Should(BeTrue())
}
func isSetupIstioReady() bool {
	list, err := gtr.cs.kubeClient.CoreV1().Pods(gtr.IstioNamespace).List(metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())
	if len(list.Items) == 0 {
		return false
	}
	nPodsGettingReady := 0
	for _, p := range list.Items {
		if p.Status.Phase == corev1.PodPending {
			nPodsGettingReady++
		}
	}
	return list.Items[0].Status.Phase == corev1.PodRunning
}

/*

- Verify that Istio is ready.
- When the pods in the `istio-system` namespace are ready or completed, you are ready to deploy the demo app.

```bash
kubectl get pods -n istio-system -w
```

- We will install the bookinfo app in the default namespace. Let's first label it for autoinjection
- This allows Istio to interface with our app.

```bash
kubectl label namespace default istio-injection=enabled
```
*/

func setupLabelAppNamespace() {
	if isSetupLabelAppNamespaceReady() {
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
	Eventually(isSetupLabelAppNamespaceReady, 80*time.Second, 250*time.Millisecond).Should(BeTrue())
}
func isSetupLabelAppNamespaceReady() bool {
	ns, err := gtr.cs.kubeClient.CoreV1().Namespaces().Get(gtr.AppNamespace, metav1.GetOptions{})
	if err != nil {
		return false
	}
	val, ok := ns.ObjectMeta.Labels["istio-injection"]
	if !ok {
		return false
	}
	return val == "enabled"
}

/*

#### Provide metric source configuration to Prometheus

Prometheus is a powerful tool for aggregating metrics. To use Prometheus most effectively, you need to tell it where it
can find metrics by specifying a list of [scrape configs](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config).

Here is an [example config](https://github.com/morvencao/istio/blob/036f689ae211cd320d68412eb42916d2debb1b73/install/kubernetes/helm/istio/charts/prometheus/templates/configmap.yaml#L15) for how Istio's metrics should be handled by Prometheus.
As you can see, scrape configs that are both insightful and resource-efficient can be quite complicated.
Additionally, managing Prometheus configs for multiple scrape targets can be difficult.

Fortunately, SuperGloo provides a powerful utility for configuring your Prometheus instance in such a way that is
appropriate for your chosen service mesh.

By default, `glooshot init` deploys an instance of Prometheus (this can be disabled).
For best results, you should configure this instance of Prometheus with the metrics that are relevant to your particular service mesh.
We will use the `supergloo set mesh stats` utility for this.

```bash
supergloo set mesh stats \
    --target-mesh glooshot.istio-istio-system \
    --prometheus-configmap glooshot.glooshot-prometheus-server
```
*/

func generateSuperglooCmd(cmdString string) *exec.Cmd {
	//cmdString = fmt.Sprintf("run gcr.io/solo-public/supergloo-cli-cloudbuild:dev1 ")
	cmd := exec.Command("supergloo", strings.Split(cmdString, " ")...)
	return cmd
}
func setupPromStats() {
	if isSetupPromStatsReady() {
		fmt.Println("skipping setup prom stats, already ready")
		return
	}
	Eventually(getIstioMeshCrd, 180*time.Second, 1*time.Second).Should(BeTrue())
	cmdString := "set mesh stats " +
		"--target-mesh glooshot.istio-istio-system " +
		"--prometheus-configmap glooshot.glooshot-prometheus-server"
	cmd := generateSuperglooCmd(cmdString)
	cmd.Stdout = GinkgoWriter
	cmd.Stderr = GinkgoWriter
	err := cmd.Run()
	Expect(err).NotTo(HaveOccurred())
	Eventually(isSetupPromStatsReady, 30*time.Second, 500*time.Millisecond).Should(BeTrue())
	Eventually(promIsStable, 60*time.Second, 500*time.Millisecond).Should(BeTrue())
}
func isSetupPromStatsReady() bool {
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
		return false
	}
	return true
}
func getIstioMeshCrd() bool {
	_, err := gtr.cs.meshClient.Read(gtr.GlooshotNamespace, gtr.tut.meshName, clients.ReadOpts{})
	if err != nil {
		return false
	}
	return true
}

/*

Note that we just had to tell SuperGloo where to find the mesh description and where to find the config map that we want to update.
SuperGloo knows which metrics are appropriate for the target mesh and sets these on the active prometheus config map.
You can find more details on setting Prometheus configurations with SuperGloo [here](https://supergloo.solo.io/tutorials/istio/tutorials-3-prometheus-metrics/).

#### Deploy the bookinfo app

- Now deploy the bookinfo app to the default namespace

```bash
kubectl apply -f https://raw.githubusercontent.com/solo-io/glooshot/master/examples/bookinfo/bookinfo.yaml
```
*/

func setupDeployBookinfo() {
	if isSetupDeployBookinfoReady() {
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
	Eventually(isSetupDeployBookinfoReady, 80*time.Second, 250*time.Millisecond).Should(BeTrue())
}
func isSetupDeployBookinfoReady() bool {
	list, err := gtr.cs.kubeClient.CoreV1().Pods(gtr.AppNamespace).List(metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())
	if len(list.Items) == 0 {
		return false
	}
	nPodsGettingReady := 0
	for _, p := range list.Items {
		if p.Status.Phase == corev1.PodPending {
			nPodsGettingReady++
		}
	}
	return list.Items[0].Status.Phase == corev1.PodRunning
}

/*

- Verify that the app is ready.
- When the pods in the `default` namespace are ready, we can start testing our app

```bash
kubectl get pods -n default -w
```

- Let's access the landing page of our app

```bash
kubectl port-forward -n default deployment/productpage-v1 9080
```

- Navigate to http://localhost:9080/productpage?u=normal in your browser.
- You should see a book description, reviews, and ratings - each provided by their respective services.
- Reload the page a few times, notice that the ratings section changes. Sometimes there are no stars, other times red or black stars appear. This is because Istio is load balancing across the four versions of the reviews service. Each reviews service renders the ratings data in a slightly different way.

- Let's use SuperGloo to modify Istio's configuration such that all reviews requests are routed to the version of the service that has red stars - and an **unknown vulnerability!**

```bash
supergloo apply routingrule trafficshifting \
    --namespace glooshot \
    --name reviews-vulnerable \
    --dest-upstreams glooshot.default-reviews-9080 \
    --target-mesh glooshot.istio-istio-system \
    --destination glooshot.default-reviews-v4-9080:1
```
*/

func setupRoutingRuleToVulnerableApp() {
	if isSetupRoutingRuleToVulnerableAppReady() {
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
	Eventually(isSetupRoutingRuleToVulnerableAppReady, 80*time.Second, 250*time.Millisecond).Should(BeTrue())
}

func isSetupRoutingRuleToVulnerableAppReady() bool {
	_, err := gtr.cs.rrClient.Read(gtr.GlooshotNamespace, gtr.tut.rrVulnerableName, clients.ReadOpts{})
	if err != nil {
		return false
	}
	return true
}

/*

- Now when you refresh the page, the stars should always be red.

- To be clear, there are four different versions of the reviews deployment. We use versions 3 and 4 in this tutorial.
- **reviews-v3** is *resilient* against cascading failures
- **reviews-v4** is *vulnerable* to cascading failures


### Create an experiment

- Create a simple experiment with `kubectl`
- We will create a fault on the ratings service such that it always returns `500` as a response code.
- We will run this experiment with the following conditions:
- The prometheus query `scalar(sum(istio_requests_total{ source_app="productpage",response_code="500"}))` must not exceed a threshold of 10.
- The experiment should expire after 600 seconds
- Execute the command below to create this experiment

```bash
cat <<EOF | kubectl apply -f -
apiVersion: glooshot.solo.io/v1
kind: Experiment
metadata:
  name: abort-ratings-metric
  namespace: default
spec:
  spec:
    duration: 600s
    failureConditions:
      - prometheusTrigger:
          customQuery: |
            scalar(sum(istio_requests_total{ source_app="productpage",response_code="500"}))
          thresholdValue: 10
          comparisonOperator: ">"
    faults:
    - destinationServices:
      - name: default-ratings-9080
        namespace: glooshot
      fault:
        abort:
          httpStatus: 500
        percentage: 100
    targetMesh:
      name: istio-istio-system
      namespace: glooshot
EOF
```
*/

func setupApplyFirstExperiment() {
	if isSetupApplyFirstExperimentReady() == readyString {
		fmt.Println("skipping setup apply first experiment, already ready")
		return
	}
	expString := `apiVersion: glooshot.solo.io/v1
kind: Experiment
metadata:
  name: abort-ratings-metric
  namespace: bookinfo
spec:
  spec:
    duration: 600s
    failureConditions:
      - trigger:
          prometheus:
            customQuery: |
              scalar(sum(rate(istio_requests_total{ source_app="productpage",response_code="500",reporter="destination",destination_app="reviews",destination_version!="v1"}[1m])))
            thresholdValue: 0.01
            comparisonOperator: ">"
    faults:
    - destinationServices:
      - name: bookinfo-ratings-9080
        namespace: glooshot
      fault:
        abort:
          httpStatus: 500
        percentage: 100
    targetMesh:
      name: istio-istio-system
      namespace: glooshot
`
	// note about the query: specify reporter to avoid duplicate reports, specify that the rate consider the last minute's worth of data
	expPath := "../../examples/bookinfo/fault-abort-ratings.yaml"
	validateFileContent(expPath, expString)
	cmdString := fmt.Sprintf("apply -f %v", expPath)
	kubectl(cmdString)
	pushCleanup(crd{"experiment", gtr.AppNamespace, "abort-ratings-metric"})
	pushCleanup(crd{"routingrule", gtr.AppNamespace, "abort-ratings-metric-0"})

	timeLimit := 15 * time.Second
	Eventually(isSetupApplyFirstExperimentReady, timeLimit, 250*time.Millisecond).Should(Equal(readyString))
}
func isSetupApplyFirstExperimentReady() string {
	if _, err := gtr.cs.expClient.Read(gtr.AppNamespace, "abort-ratings-metric", clients.ReadOpts{}); err != nil {
		return "could not read experiment"
	}
	if _, err := gtr.cs.rrClient.Read(gtr.AppNamespace, "abort-ratings-metric-0", clients.ReadOpts{}); err != nil {
		return "could not read routing rule"
	}
	return readyString
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

// for ensuring that the files we reference in our tutorials match the e2e test values
func validateFileContent(path, value string) {
	content, err := ioutil.ReadFile(path)
	Expect(err).NotTo(HaveOccurred())
	Expect(string(content)).To(Equal(value))
}

/*

- Refresh the page, you should now see a failure: none of the reviews data is rendered
- Refresh the page about 10 more times.
- Within 15 seconds after the threshold value is exceeded you should see the error go away. The experiment stop condition has been met and the fault that caused this cascading failure has been removed.
- The reason for this is that Prometheus gathers metrics every 15 seconds.
- Inspect the experiment results with the following command:

*/

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

func setupProduceTraffic() {
	portForwardApp()
	successCount := 0
	Eventually(promIsStable, 90*time.Second, 500*time.Millisecond).Should(BeTrue())
	// give prom a moment to run
	time.Sleep(1 * time.Second)
	Eventually(getNValidResponses(&successCount, 50), 60*time.Second, 25*time.Millisecond).Should(BeTrue())
	// wait for prom q's to get scraped
	Eventually(expectExpToHaveFailed(gtr.AppNamespace, "abort-ratings-metric"), 60*time.Second, 500*time.Millisecond).Should(BeNil())
	Eventually(expectExpFailureReport(gtr.AppNamespace, "abort-ratings-metric"), 15*time.Second, 250*time.Millisecond).Should(BeNil())
}
func getNValidResponses(successCount *int, targetCount int) func() bool {
	// use a closure so we can increment a success rate across retries
	return func() bool {
		_, err := curl("http://localhost:9080/productpage?u=normal")
		fmt.Printf("got %v responses\n", *successCount)
		if err == nil {
			*successCount = *successCount + 1
		}
		if *successCount > targetCount {
			return true
		}
		return false
	}
}
func expectExpToHaveFailed(namespace, name string) func() error {
	return func() error {
		exp, err := gtr.cs.expClient.Read(namespace, name, clients.ReadOpts{})
		if err != nil {
			return err
		}
		if exp.Result.State == v1.ExperimentResult_Failed {
			return nil
		}
		return fmt.Errorf("expected exp to have failed, got: %v", exp.Result.State)
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

/*
```bash
kubectl get exp abort-ratings-metric -o yaml
```

- You should see something like this:

```bash
  result:
    failureReport:
      comparison_operator: '>'
      failure_type: value_exceeded_threshold
      threshold: "10"
      value: "20"
    state: Failed
    timeFinished: "2019-05-13T17:27:49.799279861Z"
    timeStarted: "2019-05-13T17:27:34.650136785Z"
```
- Note that the state reports the experiment has "Failed". This is because the experiment was terminated because a threshold value was exceeded. If the experiment had been terminiated by a timeout, it would be in state "Succeeded".
- Experiments that fail, such as this one, indicate that our service is not as robust as we would like.
- The experiment also reports the exact value that was observed, which caused the failure. Note that the value is 20, which exceeds our limit of 10. The metric value may rise above the limit in the time it takes for Prometheus to report the exceeded limit.

### Repeat the experiment on a new version of the app
- Now that we found a weakness in our app, let's fix it.
- Let's deploy a version of the app that does not have this vulnerability. Instead of failing when no data is returned from the ratings service, the more robust version of our app will just exclude the ratings content.
- In this demo, we happened to already have deployed this version of the app. Let's use SuperGloo to update Istio so that all traffic is routed to the robust version of the app, as we did above.

```bash
kubectl delete routingrule -n glooshot reviews-vulnerable
supergloo apply routingrule trafficshifting \
    --namespace glooshot \
    --name reviews-resilient \
    --dest-upstreams glooshot.default-reviews-9080 \
    --target-mesh glooshot.istio-istio-system \
    --destination glooshot.default-reviews-v3-9080:1
```

- Verify that the new routing rule was applied
- Refresh the page, you should see no errors
- Run the following command, you should see `reviews-v3` in the `glooshot` namespace

```bash
kubectl get routingrule --all-namespaces
```

- Now let's execute this experiment again to verify that our app is robust to failure.
- This time, we do not expect any failures so we will set a shorter timeout.
- We also need to increase the threshold, since we increased our metrics in the last experiment.
- Use the following command to create a new experiment:

```bash
cat <<EOF | kubectl apply -f -
apiVersion: glooshot.solo.io/v1
kind: Experiment
metadata:
  name: abort-ratings-metric-repeat
  namespace: default
spec:
  spec:
    duration: 30s
    failureConditions:
      - prometheusTrigger:
          customQuery: |
            scalar(sum(istio_requests_total{ source_app="productpage",response_code="500"}))
          thresholdValue: 400
          comparisonOperator: ">"
    faults:
    - destinationServices:
      - name: default-ratings-9080
        namespace: glooshot
      fault:
        abort:
          httpStatus: 500
        percentage: 100
    targetMesh:
      name: istio-istio-system
      namespace: glooshot
EOF
```

- Note: for demonstration purposes, we set the threshold value to a very high number - just in case you produced a high volume
of traffic while the experiment and faulty service version were active. In a real use case it would be better to define the metrics in terms
of a rate, rather than an absolute count.

- Refresh the page, you should now see content from the reviews service and an error from the ratings service only.
- We have made our app more tolerant to failures!
- Even though the ratings service failed, the reviews service continued to fullfill its responsibilities.


- Let's inspect the experiment results:

```bash
kubectl get exp abort-ratings-metric-repeat -o yaml
```

- You should see that the experiment exceeded, after having run for the entire time limit.

```bash
  result:
    state: Succeeded
    timeFinished: "2019-05-13T18:03:05.655751554Z"
    timeStarted: "2019-05-13T18:02:35.650035732Z"
```

*/
