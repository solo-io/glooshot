# This file contains make targets related to the development workflow

# Note: when deploying remote containers during the dev-phase you should either:
# - use the -B flag
# - provide an explicit IMAGE_TAG
# Otherwise, the image tag used in the manifest will not match that of pushed image
# because the default dev-phase tag uses a timestamp.
.PHONY: deploy-manifest-dev-remote
deploy-manifest-dev-remote: docker-push render-yaml
	kubectl apply -f install/glooshot.yaml
.PHONY: deploy-manifest-dev-local
deploy-manifest-dev-local: docker render-yaml
ifeq ($(DOCKER_HOST),)
	echo "Please execute eval \`minikube docker-env\` in this shell before running: make" $@
else
	kubectl apply -f install/glooshot.yaml
endif

.PHONY: undeploy-manifest-dev
undeploy-manifest-dev:
	kubectl delete -f install/glooshot.yaml
