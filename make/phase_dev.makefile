# This file contains make targets related to the development workflow

.PHONY: eval-docker-env
eval-docker-env:
	eval `minikube docker-env`
