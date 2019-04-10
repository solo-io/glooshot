
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
