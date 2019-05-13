<h1 align="center">
    <img src="img/glooshot.png" alt="Gloo Shot" width="311" height="242">
  <br>
  Service Mesh Chaos Engineering
</h1>

Gloo Shot is a chaos engineering framework for service meshes.


[**Installation**](https://gloo.solo.io/installation/) &nbsp; |
&nbsp; [**Documentation**](https://gloo.solo.io) &nbsp; |
&nbsp; [**Blog**](https://medium.com/solo-io/) &nbsp; |
&nbsp; [**Slack**](https://slack.solo.io) &nbsp; |
&nbsp; [**Twitter**](https://twitter.com/soloio_inc)


## Summary

- [**Using Gloo Shot**](#using-gloo)
- [**What makes Gloo Shot unique**](#what-makes-gloo-unique)


## Using Gloo Shot
- **Harden your mesh**: Gloo Shot allows you to test failure modes before they occur in production.
- **Preview architectural changes**: Real deployments have different performance characteristics than your production environment. Gloo Shot allows you to simulate your productionn environment (latency, faults, etc.) prior to deployment.


## What makes Gloo Shot unique
- **Integration with the most popular service meshes**: Gloo Shot was designed for service mesh environments. It leverages [Supergloo](https://supergloo.solo.io/) for a consistent interface to multiple different service meshes.

## Getting started
- Glooshot works on top of Supergloo.
- The steps below will guide you through a complete chaos engineering session.
- Just point your `kubectl` config to the desired cluster (or `minikube`) and let's begin!
### Install Supergloo
- The latest release of `supergloo` can be found [here](https://github.com/solo-io/supergloo/releases).
- Additional details are available on the [supergloo website](https://supergloo.solo.io/installation/).
- Initialize `supergloo` and deploy Isto:
```bash
supergloo init
supergloo install istio --name istio \
  --installation-namespace istio-system \
  --mtls=true --auto-inject=true
```
### Deploy a sample app
- Here is a summary of how to get started with a sample bookstore app:
```bash
supergloo init
supergloo install istio --name istio \
  --installation-namespace istio-system \
  --mtls=true --auto-inject=true
kubectl apply -n default -f \
  https://raw.githubusercontent.com/istio/istio/1.0.6/samples/bookinfo/platform/kube/bookinfo.yaml
```
- Verify that your app has been deployed
```bash
kubectl port-forward -n default deployment/productpage-v1 9080
```
- Visit http://localhost:9080/productpage?u=test in your browser and you should see the bookstore app.
### Install Glooshot
- The latest release of `glooshot` is available [here](https://github.com/solo-io/glooshot/releases)
- Glooshot requires no setup, just define the Experiment you want to run. Let's get started with a delay:
- Define an experiment with a delay, and save it to `delay.yaml`
```yaml
apiVersion: glooshot.solo.io/v1
kind: Experiment
metadata:
  name: sample
  namespace: default
spec:
  spec:
    faults:
    - fault:
        delay:
          fixedDelay: 1s
          percentage: 100
      service:
        upstream:
          name: todo
          namespace: default
    stopCondition:
      duration: 60s
      metric:
      - metricName: dinner
        value: 1800

```
- Now create that resource with:
```bash
glooshot apply -f delay.yaml
```
- Verify that it has been applied with:
```bash
glooshot get experiments
```

## Next Steps
- Join us on our slack channel: [https://slack.solo.io/](https://slack.solo.io/)
- Follow us on Twitter: [https://twitter.com/soloio_inc](https://twitter.com/soloio_inc)
- Check out the docs: [https://gloo.solo.io](https://gloo.solo.io)
- Check out the code and contribute: [Contribution Guide](CONTRIBUTING.md)
- Contribute to the [Docs](https://github.com/solo-io/solo-docs)

### Thanks

**Gloo Shot** would not be possible without the valuable open-source work of projects in the community. We would like to extend a special thank-you to [Envoy](https://www.envoyproxy.io).

