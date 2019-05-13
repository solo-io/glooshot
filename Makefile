#----------------------------------------------------------------------------------
# This portion is managed by github.com/solo-io/build
#----------------------------------------------------------------------------------
# NOTE! All make targets that use the computed values must depend on the "must"
# target to ensure the expected computed values were recieved
.PHONY: must
must: validate-computed-values

# Read computed values into variables that can be used by make
# Since both stdout and stderr are passed, our make targets validate the variables
BUILD_CONFIG_FILE ?= solo-project.yaml
BUILD_CMD := SOLOBUILD_CONFIG_FILE=${BUILD_CONFIG_FILE} go run cmd/build/main.go
RELEASE := $(shell ${BUILD_CMD} parse-env release)
VERSION := $(shell ${BUILD_CMD} parse-env version)
IMAGE_TAG := $(shell ${BUILD_CMD} parse-env image-tag)
CONTAINER_REPO_ORG := $(shell ${BUILD_CMD} parse-env container-prefix)
HELM_REPO := $(shell ${BUILD_CMD} parse-env helm-repo)

# use this, or the shorter alias "must", as a dependency for any target that uses
# values produced by the build tool
.PHONY: validate-computed-values
validate-computed-values:
	${BUILD_CMD} validate-operating-parameters \
		$(RELEASE) \
		$(VERSION) \
		$(CONTAINER_REPO_ORG) \
		$(IMAGE_TAG) \
		$(HELM_REPO)

.PHONY: preview-computed-values
preview-computed-values: must
	echo summary of computed values - \
		release: $(RELEASE), \
		version: $(VERSION), \
		container-prefix: $(CONTAINER_REPO_ORG), \
		image-tag: $(IMAGE_TAG), \
		helm-repo: $(HELM_REPO)

#### END OF MANAGED PORTION




SOLO_NAME := glooshot
ROOTDIR := $(shell pwd)
OUTPUT_DIR ?= $(ROOTDIR)/_output

FORMAT_DIRS = ./pkg/ ./cmd/ ./ci/
SOURCES := $(shell find . -name "*.go" | grep -v test.go | grep -v '\.\#*' | grep -v mock)
LDFLAGS := "-X github.com/solo-io/$(SOLO_NAME)/pkg/version.Version=$(VERSION)"
GCFLAGS := all="-N -l"


#----------------------------------------------------------------------------------
# glooshot - main
#----------------------------------------------------------------------------------

.PHONY: glooshot
glooshot: must glooshot-cli glooshot-operator

.PHONY: glooshot-docker
glooshot-docker: must $(OUTPUT_DIR)/glooshot-docker

.PHONY: glooshot-docker-push
glooshot-docker-push: must glooshot-docker
	docker push $(CONTAINER_REPO_ORG)/$(GLOOSHOT_OPERATOR_NAME):$(IMAGE_TAG)

#----------------------------------------------------------------------------------
# glooshot CLI
#----------------------------------------------------------------------------------
GLOOSHOT_CLI_NAME=glooshot
GLOOSHOT_CLI_DIR=cmd/cli
GLOOSHOT_CLI_SOURCES=$(shell find $(GLOOSHOT_CLI_DIR) -name "*.go" | grep -v test | grep -v generated.go)

.PHONY: glooshot-cli
glooshot-cli: must $(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-linux-amd64 $(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-darwin-amd64 $(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-windows-amd64.exe

$(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-linux-amd64: $(GLOOSHOT_CLI_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_CLI_DIR)/main.go

$(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-darwin-amd64: $(GLOOSHOT_CLI_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_CLI_DIR)/main.go

$(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-windows-amd64.exe: $(GLOOSHOT_CLI_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_CLI_DIR)/main.go

#----------------------------------------------------------------------------------
# glooshot OPERATOR
#----------------------------------------------------------------------------------
GLOOSHOT_OPERATOR_NAME=glooshot-op
GLOOSHOT_OPERATOR_DIR=cmd/glooshot
GLOOSHOT_OPERATOR_SOURCES=$(shell find $(GLOOSHOT_OPERATOR_DIR) -name "*.go" | grep -v test | grep -v generated.go)

$(OUTPUT_DIR)/$(GLOOSHOT_OPERATOR_NAME)-linux-amd64: $(GLOOSHOT_OPERATOR_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_OPERATOR_DIR)/main.go

# Make a darwin binary for local testing on mac. Not for distribution.
$(OUTPUT_DIR)/$(GLOOSHOT_OPERATOR_NAME)-darwin: $(GLOOSHOT_OPERATOR_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_OPERATOR_DIR)/main.go

.PHONY: glooshot-operator
glooshot-operator: must $(OUTPUT_DIR)/$(GLOOSHOT_OPERATOR_NAME)-linux-amd64 $(OUTPUT_DIR)/$(GLOOSHOT_OPERATOR_NAME)-darwin

# copy the docker file into the build dir
$(OUTPUT_DIR)/Dockerfile.glooshot: $(GLOOSHOT_OPERATOR_DIR)/Dockerfile
	cp $< $@

$(OUTPUT_DIR)/glooshot-docker: glooshot-operator $(OUTPUT_DIR)/Dockerfile.glooshot
	docker build -t $(CONTAINER_REPO_ORG)/$(GLOOSHOT_OPERATOR_NAME):$(IMAGE_TAG) $(OUTPUT_DIR) -f $(OUTPUT_DIR)/Dockerfile.glooshot
	touch $@

#----------------------------------------------------------------------------------
# Deployment Manifests / Helm
#----------------------------------------------------------------------------------

HELM_SYNC_DIR := $(OUTPUT_DIR)/helm
HELM_DIR := install/helm/$(SOLO_NAME)
INSTALL_NAMESPACE ?= $(SOLO_NAME)

.PHONY: manifest
manifest: prepare-helm init-helm install/$(SOLO_NAME).yaml update-helm-chart

# creates Chart.yaml, values.yaml
.PHONY: prepare-helm
prepare-helm: must
	go run install/helm/$(SOLO_NAME)/generate/cmd/generate.go $(IMAGE_TAG) $(CONTAINER_REPO_ORG)

.PHONY: init-helm
init-helm:
	helm repo add supergloo https://storage.googleapis.com/supergloo-helm
	helm dependency update install/helm/glooshot

.PHONY: update-helm-chart
update-helm-chart: must
	mkdir -p $(HELM_SYNC_DIR)/charts
	helm package --destination $(HELM_SYNC_DIR)/charts $(HELM_DIR)
	helm repo index $(HELM_SYNC_DIR)

HELMFLAGS := --namespace $(INSTALL_NAMESPACE) --set namespace.create=true

install/$(SOLO_NAME).yaml: prepare-helm init-helm
	helm template install/helm/$(SOLO_NAME) $(HELMFLAGS) > $@

.PHONY: render-yaml
render-yaml: must install/$(SOLO_NAME).yaml

#----------------------------------------------------------------------------------
# MAIN TARGETS
#----------------------------------------------------------------------------------

.PHONY: docker
docker: must glooshot-cli glooshot-operator glooshot-docker

.PHONY: docker-push
docker-push: must docker glooshot-docker-push

.PHONY: release
release: must render-yaml docker-push
# note, this only releases when TAGGED_VERSION has been set
	go run ci/upload_github_release_assets.go









#----------------------------------------------------------------------------------
# Common config, avoid editing the below targets directly
#----------------------------------------------------------------------------------

#----------------------------------------------------------------------------------
# Repo setup
#----------------------------------------------------------------------------------
# https://www.viget.com/articles/two-ways-to-share-git-hooks-with-your-team/
.PHONY: init
init:
	git config core.hooksPath .githooks

.PHONY: pin-repos
pin-repos:
	go run ci/pin_repos.go

.PHONY: update-deps
update-deps:
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/gogo/protobuf/gogoproto
	go get -u github.com/gogo/protobuf/protoc-gen-gogo
	go get -u github.com/envoyproxy/protoc-gen-validate
	go get -u github.com/paulvollmer/2gobytes
	go get -v -u github.com/golang/mock/gomock
	go get -v -u github.com/golang/mock/mockgen
	go install github.com/golang/mock/mockgen

#----------------------------------------------------------------------------------
# Generated Code and Docs
#----------------------------------------------------------------------------------

.PHONY: generated-code
generated-code: must $(OUTPUT_DIR)/.generated-code

SUBDIRS:=pkg cmd ci
$(OUTPUT_DIR)/.generated-code:
	go generate ./...
	gofmt -w $(SUBDIRS)
	goimports -w $(SUBDIRS)
	mkdir -p $(OUTPUT_DIR)
	touch $@

#----------------------------------------------------------------------------------
# Checks
#----------------------------------------------------------------------------------

.PHONY: check-format
check-format: must
	NOT_FORMATTED=$$(gofmt -l $(FORMAT_DIRS)) && if [ -n "$$NOT_FORMATTED" ]; then echo These files are not formatted: $$NOT_FORMATTED; exit 1; fi

# TODO - enable spell check
