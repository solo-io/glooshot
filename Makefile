SOLO_NAME := glooshot
ROOTDIR := $(shell pwd)
OUTPUT_DIR ?= $(ROOTDIR)/_output

FORMAT_DIRS = ./pkg/ ./cmd/ ./ci/
SOURCES := $(shell find . -name "*.go" | grep -v test.go | grep -v '\.\#*')
LDFLAGS := "-X github.com/solo-io/$(SOLO_NAME)/pkg/version.Version=$(VERSION)"
GCFLAGS := all="-N -l"

#-------------------------------------------------------------------------------
# Establish container values
#-------------------------------------------------------------------------------
# Passed by cloudbuild
GCLOUD_PROJECT_ID := $(GCLOUD_PROJECT_ID)
VERSION ?= $(shell echo $(TAGGED_VERSION) | cut -c 2-)
LAST_COMMIT = $(shell git rev-parse HEAD | cut -c 1-6)
# Note: need to evaluate this with := to avoid re-evaluation
STAMP_DDHHMMSS := $(shell date +%d%H%M%S)
IMAGE_TAG ?= $(LAST_COMMIT)-$(STAMP_DDHHMMSS)-pre
CONTAINER_REPO_ORG ?= gcr.io/$(GCLOUD_PROJECT_ID)

ifeq ($(TAGGED_VERSION),)
# no tagged version provided, we are not in a release, use CI values
# USE CI VALUES, overridable, above
else
# a tagged version has been provided, we are performing a release
# USE RELEASE VALUES, hard-coded below
  CONTAINER_ORG = soloio
# Use docker repo, which is inferred when none provided
  CONTAINER_REPO_ORG=$(CONTAINER_ORG)
  IMAGE_TAG = $(VERSION)
endif


#----------------------------------------------------------------------------------
# glooshot - main
#----------------------------------------------------------------------------------

.PHONY: glooshot
glooshot: glooshot-cli glooshot-operator

.PHONY: glooshot-docker
glooshot-docker: $(OUTPUT_DIR)/glooshot-docker

.PHONY: glooshot-docker-push
glooshot-docker-push: glooshot-docker
	docker push $(CONTAINER_REPO_ORG)/$(GLOOSHOT_OPERATOR_NAME):$(IMAGE_TAG)

#----------------------------------------------------------------------------------
# glooshot CLI
#----------------------------------------------------------------------------------
GLOOSHOT_CLI_NAME=glooshot
GLOOSHOT_CLI_DIR=cmd/cli
GLOOSHOT_CLI_SOURCES=$(shell find $(GLOOSHOT_CLI_DIR) -name "*.go" | grep -v test | grep -v generated.go)

.PHONY: glooshot-cli
glooshot-cli: $(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-linux-amd64 $(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-darwin-amd64 $(OUTPUT_DIR)/$(GLOOSHOT_CLI_NAME)-windows-amd64.exe

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
glooshot-operator: $(OUTPUT_DIR)/$(GLOOSHOT_OPERATOR_NAME)-linux-amd64 $(OUTPUT_DIR)/$(GLOOSHOT_OPERATOR_NAME)-darwin

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
manifest: prepare-helm install/$(SOLO_NAME).yaml update-helm-chart

# creates Chart.yaml, values.yaml
.PHONY: prepare-helm
prepare-helm:
	go run install/helm/$(SOLO_NAME)/generate/cmd/generate.go $(IMAGE_TAG) $(CONTAINER_REPO_ORG)

.PHONY: update-helm-chart
update-helm-chart:
	mkdir -p $(HELM_SYNC_DIR)/charts
	helm package --destination $(HELM_SYNC_DIR)/charts $(HELM_DIR)
	helm repo index $(HELM_SYNC_DIR)

HELMFLAGS := --namespace $(INSTALL_NAMESPACE) --set namespace.create=true

install/$(SOLO_NAME).yaml: prepare-helm
	helm template install/helm/$(SOLO_NAME) $(HELMFLAGS) > $@

.PHONY: render-yaml
render-yaml: install/$(SOLO_NAME).yaml

#----------------------------------------------------------------------------------
# MAIN TARGETS
#----------------------------------------------------------------------------------

.PHONY: docker
docker: glooshot-cli glooshot-operator glooshot-docker

.PHONY: docker-push
docker-push: docker glooshot-docker-push

.PHONY: release
release: render-yaml docker-push
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

#----------------------------------------------------------------------------------
# Generated Code and Docs
#----------------------------------------------------------------------------------

.PHONY: generated-code
generated-code: $(OUTPUT_DIR)/.generated-code

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
check-format:
	NOT_FORMATTED=$$(gofmt -l $(FORMAT_DIRS)) && if [ -n "$$NOT_FORMATTED" ]; then echo These files are not formatted: $$NOT_FORMATTED; exit 1; fi

# TODO - enable spell check
