
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
make glooshot-docker
kubectl apply -f install/glooshot.yaml
```


## Building and deploying to an external cluster
```bash
export VERSION=mkdev1
export TAGGED_VERSION=mkdev1
make render-yaml -B
```


# Enable Stats
- To enable stats, set the env var `START_STATS_SERVER=1`
- To view stats, access your service's port `9091`
  - If running locally, that will be `localhost:9091`
