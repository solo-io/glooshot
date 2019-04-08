# Base
#----------------------------------------------------------------------------------

SOLO_NAME := gloo-shot
SOLO_SHORT_NAME := gs
ROOTDIR := $(shell pwd)
OUTPUT_DIR ?= $(ROOTDIR)/_output
SOURCES := $(shell find . -name "*.go" | grep -v test.go | grep -v '\.\#*')
RELEASE := "true"
ifeq ($(TAGGED_VERSION),)
	# TAGGED_VERSION := $(shell git describe --tags)
	# This doesn't work in CI, need to find another way...
	TAGGED_VERSION := vdev
	RELEASE := "false"
endif
VERSION ?= $(shell echo $(TAGGED_VERSION) | cut -c 2-)

LDFLAGS := "-X github.com/solo-io/$(SOLO_NAME)/services/internal/version.Version=$(VERSION)"
GCFLAGS := all="-N -l"

# Passed by cloudbuild
GCLOUD_PROJECT_ID := $(GCLOUD_PROJECT_ID)
BUILD_ID := $(BUILD_ID)

TEST_IMAGE_TAG := test-$(BUILD_ID)
TEST_ASSET_DIR := $(ROOTDIR)/_test
GCR_REPO_PREFIX := gcr.io/$(GCLOUD_PROJECT_ID)

#----------------------------------------------------------------------------------
# Repo setup
#----------------------------------------------------------------------------------

# https://www.viget.com/articles/two-ways-to-share-git-hooks-with-your-team/
.PHONY: init
init:
	git config core.hooksPath .githooks

.PHONY: update-deps
update-deps:
	go get -u golang.org/x/tools/cmd/goimports
	go get -u github.com/gogo/protobuf/gogoproto
	go get -u github.com/gogo/protobuf/protoc-gen-gogo
	go get -u github.com/envoyproxy/protoc-gen-validate
	go get -u github.com/paulvollmer/2gobytes

.PHONY: pin-repos
pin-repos:
	go run ci/pin_repos.go

.PHONY: check-format
check-format:
	NOT_FORMATTED=$$(gofmt -l ./services/ ./ci/) && if [ -n "$$NOT_FORMATTED" ]; then echo These files are not formatted: $$NOT_FORMATTED; exit 1; fi

.PHONY: check-spelling
check-spelling:
	./ci/spell.sh check

#----------------------------------------------------------------------------------
# Generated Code and Docs
#----------------------------------------------------------------------------------

.PHONY: generated-code
generated-code: $(OUTPUT_DIR)/.generated-code

SUBDIRS:=services ci
$(OUTPUT_DIR)/.generated-code:
	go generate ./...
	gofmt -w $(SUBDIRS)
	goimports -w $(SUBDIRS)
	mkdir -p $(OUTPUT_DIR)
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
prepare-helm:
	go run install/helm/$(SOLO_NAME)/generate/cmd/generate.go $(VERSION)

update-helm-chart:
ifeq ($(RELEASE),"true")
	mkdir -p $(HELM_SYNC_DIR)/charts
	helm package --destination $(HELM_SYNC_DIR)/charts $(HELM_DIR)
	helm repo index $(HELM_SYNC_DIR)
endif

HELMFLAGS := --namespace $(INSTALL_NAMESPACE) --set namespace.create=true

install/$(SOLO_NAME).yaml: prepare-helm
	helm template install/helm/$(SOLO_NAME) $(HELMFLAGS) > $@

.PHONY: render-yaml
render-yaml: install/$(SOLO_NAME).yaml

#----------------------------------------------------------------------------------
# Apiserver
#----------------------------------------------------------------------------------

APISERVER_DIR=services/apiserver
APISERVER_SOURCES=$(shell find $(APISERVER_DIR) -name "*.go" | grep -v test | grep -v generated.go)

$(OUTPUT_DIR)/$(SOLO_SHORT_NAME)-apiserver-linux-amd64: $(APISERVER_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(APISERVER_DIR)/cmd/main.go

.PHONY: apiserver
apiserver: $(OUTPUT_DIR)/$(SOLO_SHORT_NAME)-apiserver-linux-amd64

$(OUTPUT_DIR)/Dockerfile.apiserver: $(APISERVER_DIR)/cmd/Dockerfile
	cp $< $@

.PHONY: apiserver-docker
apiserver-docker: $(OUTPUT_DIR)/.apiserver-docker

$(OUTPUT_DIR)/.apiserver-docker: $(OUTPUT_DIR)/$(SOLO_SHORT_NAME)-apiserver-linux-amd64 $(OUTPUT_DIR)/Dockerfile.apiserver
	docker build -t soloio/$(SOLO_SHORT_NAME)-apiserver:$(VERSION) $(call get_test_tag_option,$(SOLO_SHORT_NAME)-apiserver) $(OUTPUT_DIR) -f $(OUTPUT_DIR)/Dockerfile.apiserver
	touch $@

#----------------------------------------------------------------------------------
# Operator
#----------------------------------------------------------------------------------

OPERATOR_DIR=services/operator
OPERATOR_SOURCES=$(shell find $(OPERATOR_DIR) -name "*.go" | grep -v test | grep -v generated.go)

$(OUTPUT_DIR)/$(SOLO_SHORT_NAME)-operator-linux-amd64: $(OPERATOR_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(OPERATOR_DIR)/cmd/main.go

.PHONY: operator
operator: $(OUTPUT_DIR)/$(SOLO_SHORT_NAME)-operator-linux-amd64

$(OUTPUT_DIR)/Dockerfile.operator: $(OPERATOR_DIR)/cmd/Dockerfile
	cp $< $@

.PHONY: operator-docker
operator-docker: $(OUTPUT_DIR)/.operator-docker

$(OUTPUT_DIR)/.operator-docker: $(OUTPUT_DIR)/$(SOLO_SHORT_NAME)-operator-linux-amd64 $(OUTPUT_DIR)/Dockerfile.operator
	docker build -t soloio/$(SOLO_SHORT_NAME)-operator:$(VERSION) $(call get_test_tag_option,$(SOLO_SHORT_NAME)-operator) $(OUTPUT_DIR) -f $(OUTPUT_DIR)/Dockerfile.operator
	touch $@

#----------------------------------------------------------------------------------
# Release
#----------------------------------------------------------------------------------

.PHONY: upload-github-release-assets
upload-github-release-assets: render-yaml
	go run ci/upload_github_release_assets.go

.PHONY: release
release: docker-push upload-github-release-assets

#----------------------------------------------------------------------------------
# Docker
#----------------------------------------------------------------------------------
#
#---------
#--------- Push
#---------

DOCKER_IMAGES :=
ifeq ($(RELEASE),"true")
	DOCKER_IMAGES := docker
endif

.PHONY: docker docker-push
docker: apiserver-docker operator-docker

# Depends on DOCKER_IMAGES, which is set to docker if RELEASE is "true", otherwise empty (making this a no-op).
# This prevents executing the dependent targets if RELEASE is not true, while still enabling `make docker`
# to be used for local testing.
# docker-push is intended to be run by CI
docker-push: $(DOCKER_IMAGES)
ifeq ($(RELEASE),"true")
	docker push soloio/$(SOLO_SHORT_NAME)-operator:$(VERSION) && \
	docker push soloio/$(SOLO_SHORT_NAME)-apiserver:$(VERSION)
endif

push-kind-images: docker
	kind load docker-image soloio/$(SOLO_SHORT_NAME)-operator:$(VERSION) --name $(CLUSTER_NAME) && \
	kind load docker-image soloio/$(SOLO_SHORT_NAME)-apiserver:$(VERSION) --name $(CLUSTER_NAME)
