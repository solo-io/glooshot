
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
BUILD_ID=my-tag make docker-push render-yaml -B
kubectl apply -f install/glooshot.yaml
```


# Enable Stats
- To enable stats, set the env var `START_STATS_SERVER=1`
- To view stats, access your service's port `9091`
  - If running locally, that will be `localhost:9091`
  
# Change build parameters
- Many `make` targets read values from the `solo-projects.yaml` build config file in the root directory. You can
override these values by creating a new configuration file and specifying it as follows:
```bash
BUILD_CONFIG_FILE=my-custom-build-config.yaml make render-yaml -B
```
