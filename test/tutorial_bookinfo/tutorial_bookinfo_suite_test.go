package tutorial_bookinfo

import (
	"fmt"
	"os"
	"testing"

	buildv1 "github.com/solo-io/build/pkg/api/v1"
	"github.com/solo-io/build/pkg/ingest"

	"github.com/solo-io/glooshot/pkg/cli"
	"github.com/spf13/cobra"

	"github.com/solo-io/go-utils/testutils"

	"github.com/solo-io/solo-kit/test/helpers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestTutorialBookinfo(t *testing.T) {

	helpers.RegisterPreFailHandler(
		func() {
			testutils.PrintTrimmedStack()
		})
	helpers.RegisterCommonFailHandlers()
	helpers.SetupLog()
	RunSpecs(t, "Bookinfo Tutorial Suite")
}

var _ = BeforeSuite(func() {
	// set up the cluster
	if os.Getenv("CI_TESTS") == "1" {
		fmt.Printf("this test is disabled in CI. to run, ensure env var `CI_TESTS` is not set to 1")
		return
	}

	buildRun, err := ingest.InitializeBuildRun("../../solo-project.yaml", &buildv1.BuildEnvVars{})
	Expect(err).NotTo(HaveOccurred())
	tp := testParams{buildId: buildRun.Config.BuildEnvVars.BuildId}
	setupCluster(tp)

})

func newTestResources() testResources {
	return testResources{}
}

type testResources struct {
	glooshotCli *cobra.Command
}

type testParams struct {
	buildId string
}

func setupCluster(tp testParams) {
	setupGlooshotInit(tp)

}

/*
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

- `glooshot` [(download)](https://github.com/solo-io/glooshot/releases) command line tool, v0.0.4 or greater
- `supergloo` [(download)](https://supergloo.solo.io/installation/) command line tool, v0.3.18 or greater, for simplified mesh management during the tutorial.
- `kubectl` [(download)](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
- A Kubernetes cluster - [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/#install-minikube) will do

### Setup

#### Deploy Gloo Shot

- Gloo Shot can easily be deployed from the command line tool.
- This will put Gloo Shot in the `glooshot` namespace.

```bash
glooshot init
```
*/

func setupGlooshotInit(tp testParams) {

	out, err := cli.GlooshotConfig.RunForTest(fmt.Sprintf("init -f ../../_output/helm/charts/glooshot-%v.tgz", tp.buildId))
	Expect(err).NotTo(HaveOccurred())
	//Expect(out.CobraStderr).To(Equal(""))
	fmt.Println(out.CobraStderr)
	fmt.Println(out.CobraStdout)

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

Note that we just had to tell SuperGloo where to find the mesh description and where to find the config map that we want to update.
SuperGloo knows which metrics are appropriate for the target mesh and sets these on the active prometheus config map.
You can find more details on setting Prometheus configurations with SuperGloo [here](https://supergloo.solo.io/tutorials/istio/tutorials-3-prometheus-metrics/).

#### Deploy the bookinfo app

- Now deploy the bookinfo app to the default namespace

```bash
kubectl apply -f https://raw.githubusercontent.com/solo-io/glooshot/master/examples/bookinfo/bookinfo.yaml
```

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

- Refresh the page, you should now see a failure: none of the reviews data is rendered
- Refresh the page about 10 more times.
- Within 15 seconds after the threshold value is exceeded you should see the error go away. The experiment stop condition has been met and the fault that caused this cascading failure has been removed.
- The reason for this is that Prometheus gathers metrics every 15 seconds.
- Inspect the experiment results with the following command:

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
