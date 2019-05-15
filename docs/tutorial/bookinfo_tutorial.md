# Bookinfo Tutorial

This tutorial will show you how to use Glooshot to apply chaos experiments to a simple service mesh app.
We will use a slight modification of the familiar bookinfo app from Istio's
[sample app repo](https://github.com/istio/istio/tree/master/samples/bookinfo). We have modified the reviews service to
include a vulnerability that can lead to cascading failure. We will use Glooshot to detect this weakness.


### The Goal

Services should be resilient to system outages. In this example, we show how to detect cascading failures: failures
where an error in one service disables other services that interact with it. In the diagram below, we show two versions
of a reviews service. The version on the top right fails when it does not receive a valid response from the ratings.
The version on the bottom right handles the error more gracefully. It still provides review information even though the
ratings data is not available.

![bookinfo resilience overview](./bookinfo_resilience_demo.png "bookinfo reslience demo")

## Prerequisites

To follow this demo, you will need the following:
- glooshot v0.0.2 or greater [(download)](https://github.com/solo-io/glooshot/releases)
- [Supergloo](supergloo.solo.io) v0.3.18 or greater [(download)](https://github.com/solo-io/supergloo/releases)
- A Kubernetes cluster - [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/#install-minikube) will do

## Setup

### Deploy Glooshot

- Glooshot can easily be deployed from the command line tool.
  - This will put Glooshot in the `glooshot` namespace.
```bash
glooshot init
```

### Install a service mesh (if you have not already)

- Install a service mesh.
  - We will use Istio for this tutorial.
  - We will use Supergloo to install Istio with Prometheus.

```bash
supergloo install istio \
    --namespace glooshot \
    --name istio \
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

### Deploy the bookinfo app

- Now deploy the bookinfo app to the default namespace
```bash
kubectl apply -f bookinfo.yaml
```

- Verify that the app is ready.
  - When the pods in the `default` namespace are ready, we can start testing our app
```bash
kubectl get pods -n default -w
```

- Let's access the landing page of our app
  - Execute the command below.
  - Navigate to http://localhost:9080/productpage?u=normal in your browser.
  - You should see a book description, reviews, and ratings - each provided by their respective services.
  - Reload the page a few times, notice that the ratings section changes. Sometimes there are no stars, other times red or black stars appear.. This is because Istio is load balancing across the four versions of the reviews service. Each reviews service renders the ratings data in a slightly different way.
```bash
kubectl port-forward -n default deployment/productpage-v1 9080
```

- Let's use Supergloo to modify Istio's configuration such that all reviews requests are routed to the version of the service that has red stars - and an **unknown vulnerability!**
  - Execute the command below
  - Now when you refresh the page, the stars should always be red.
```bash
supergloo apply routingrule trafficshifting \
    --name reviews-v4 \
    --dest-upstreams supergloo-system.default-reviews-9080 \
    --target-mesh supergloo-system.istio \
    --destination supergloo-system.default-reviews-v4-9080:1
```


## Create an experiment

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
        namespace: supergloo-system
      fault:
        abort:
          httpStatus: 500
        percentage: 100
    targetMesh:
      name: istio
      namespace: supergloo-system
EOF
```

- Refresh the page, you should now see a failure: none of the reviews data is rendered
- Refresh the page about 10 more times.
- Within 15 seconds after the threshold value is exceeded you should see the error go away. The experiment stop condition has been met and the fault that caused this cascading failure has been removed.
- Inspect the experiment results with the following command:
```bash
k get exp abort-ratings-metric -o yaml
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
- The experiment also reports the exact value that was observed, which caused the failure. Note that the value is 20, which exceeds our limit of 10. The reason for this is that Prometheus gathers metrics every 15 seconds. The metric value may rise above the limit in the time it takes for Prometheus to report the exceeded limit.

## Repeat the experiment on a new version of the app
- Now that we found a weakness in our app, let's fix it.
- Let's deploy a version of the app that does not have this vulnerability. Instead of failing when no data is returned from the ratings service, the more robust version of our app will just exclude the ratings content.
- In this demo, we happened to already have deployed this version of the app. Let's use Supergloo to update Istio so that all traffic is routed to the robust version of the app, as we did above.
```bash
kubectl delete routingrule -n supergloo-system reviews-v4
supergloo apply routingrule trafficshifting \
          --name reviews-v3 \
          --dest-upstreams supergloo-system.default-reviews-9080 \
          --target-mesh supergloo-system.istio \
          --destination supergloo-system.default-reviews-v3-9080:1
```

- Verify that the new routing rule was applied
  - Refresh the page, you should see no errors
  - Run the following command, you should see `reviews-v3` in the `supergloo-system` namespace
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
          thresholdValue: 60
          comparisonOperator: ">"
    faults:
    - destinationServices:
      - name: default-ratings-9080
        namespace: supergloo-system
      fault:
        abort:
          httpStatus: 500
        percentage: 100
    targetMesh:
      name: istio
      namespace: supergloo-system
EOF
```

- Refresh the page, you should now see content from the reviews service and an error from the ratings service only.
- We have made our app more tolerant to failures!
  - Even though the ratings service failed, the reviews service continued to fullfill its responsibilities.


- Let's inspect the experiment results:
```bash
kubectl get exp abort-ratings-metric -o yaml
```

- You should see that the experiment exceeded, after having run for the entire time limit.
```bash
  result:
    state: Succeeded
    timeFinished: "2019-05-13T18:03:05.655751554Z"
    timeStarted: "2019-05-13T18:02:35.650035732Z"
```


