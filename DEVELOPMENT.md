
# Development workflow for testing changes to Glooshot
## Building and deploying resources locally
- initialize repo
```
git clone https://github.com/solo-io/glooshot
```
- any time you make changes
```
make pin-repos
make update-deps
dep ensure -v
make prepare-helm
make render-yaml
```
- test changes in minikube
```
eval `minikube docker-env`
make deploy-manifest-dev-local -B
# test, then remove with:
make undeploy-manifest-dev
```


## Building and deploying to an external cluster
```bash
export CONTAINER_REPO="myrepo.com" # optional
export CONTAINER_ORG="myorg" # optional
make docker-push render-yaml -B
kubectl apply -f install/glooshot.yaml
```


# Enable Stats
- To enable stats, set the env var `START_STATS_SERVER=1`
- To view stats, access your service's port `9091`
  - If running locally, that will be `localhost:9091`
