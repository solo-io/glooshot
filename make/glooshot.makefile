#----------------------------------------------------------------------------------
# glooshot
# consists of:
# - cli - builds binary
# - in-cluster process - builds binary and docker image
#----------------------------------------------------------------------------------

.PHONY: glooshot-cli
glooshot: glooshot-cli glooshot-operator

#----------------------------------------------------------------------------------
# CLI
#----------------------------------------------------------------------------------
GLOOSHOT_CLI_NAME=cmd/glooshot
GLOOSHOT_CLI_DIR=cmd/glooshot
GLOOSHOT_CLI_SOURCES=$(shell find $(GLOOSHOT_CLI_DIR) -name "*.go" | grep -v test | grep -v generated.go)

.PHONY: glooshot-cli
glooshot_cli: $(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-linux-amd64 $(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-darwin

$(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-linux-amd64: $(GLOOSHOT_CLI_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_CLI_DIR)/main.go

$(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-darwin: $(GLOOSHOT_CLI_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_CLI_DIR)/main.go


#----------------------------------------------------------------------------------
# OPERATOR
#----------------------------------------------------------------------------------
GLOOSHOT_OPERATOR_NAME=cmd/glooshotop
GLOOSHOT_OPERATOR_DIR=cmd/glooshot
GLOOSHOT_OPERATOR_SOURCES=$(shell find $(GLOOSHOT_OPERATOR_DIR) -name "*.go" | grep -v test | grep -v generated.go)

$(OUTPUT_DIR)/$(GLOOSHOT_OPERATOR_NAME)-linux-amd64: $(GLOOSHOT_OPERATOR_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_OPERATOR_DIR)/main.go

# Make a darwin binary for local testing on mac. Not for distribution.
$(OUTPUT_DIR)/$(GLOOSHOT_OPERATOR_NAME)-darwin: $(GLOOSHOT_OPERATOR_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_OPERATOR_DIR)/main.go

.PHONY: glooshot-operator
glooshot-operator: $(OUTPUT_DIR)/$(GLOOSHOT_OPERATOR_NAME)-linux-amd64 $(OUTPUT_DIR)/$(GLOOSHOT_OPERATOR_NAME)-darwin

# copy the docker file into the build dir
$(OUTPUT_DIR)/Dockerfile.glooshot: $(GLOOSHOT_OPERATOR_DIR)/Dockerfile
	cp $< $@

.PHONY: glooshot-docker
glooshot-docker: $(OUTPUT_DIR)/glooshot-docker

$(OUTPUT_DIR)/glooshot-docker: glooshot-operator $(OUTPUT_DIR)/Dockerfile.glooshot
	docker build -t $(CONTAINER_REPO_ORG)/$(GLOOSHOT_OPERATOR_NAME):$(IMAGE_TAG) $(OUTPUT_DIR) -f $(OUTPUT_DIR)/Dockerfile.glooshot
	touch $@
