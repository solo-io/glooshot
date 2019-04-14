
#----------------------------------------------------------------------------------
# Docker
#----------------------------------------------------------------------------------

.PHONY: docker docker-push
docker: glooshot-cli glooshot-operator glooshot-docker

docker-push: docker
	docker push $(CONTAINER_REPO_ORG)/$(GLOOSHOT_OPERATOR_NAME):$(IMAGE_TAG)
