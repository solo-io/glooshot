SOLO_NAME := glooshot
ROOTDIR := $(shell pwd)
OUTPUT_DIR ?= $(ROOTDIR)/_output

#--------------------------- Determine Phase ----------------------------------#
# This makefile is oriented around development lifecycle "phases"
# phases include:
# - dev: any local builds
# - buildtest: builds in CI, excluding releases
# - release: builds in CI for releases
PHASE := "dev"
# Passed by cloudbuild
GCLOUD_PROJECT_ID := $(GCLOUD_PROJECT_ID)
# Determine lifecycle phase
ifeq ($(TAGGED_VERSION),)
  TAGGED_VERSION := vdev
  ifeq ($(GCLOUD_PROJECT_ID),)
    # not inside CI
    PHASE = "dev"
  else
    # inside CI, but not making a release
    PHASE = "buildtest"
  endif
else
  # a tagged version has been provided, we are performing a relase
  PHASE = "release"
endif

#---------------- Compute phase-specific and phase-configurable values ---------#
VERSION ?= $(shell echo $(TAGGED_VERSION) | cut -c 2-)
LAST_COMMIT = $(shell git rev-parse HEAD | cut -c 1-6)

CONTAINER_ORG ?= soloio
# CONTAINER_REPO := $(CONTAINER_REPO) # defaults to docker
# the docker documentation states the implied repo url is: registry-1.docker.io
# https://docs.docker.com/engine/reference/commandline/tag/
# just in case, we will let the docker tool provide that
GCR_REPO_PREFIX := gcr.io/$(GCLOUD_PROJECT_ID)

# Note: need to evaluate this with := to avoid re-evaluation
STAMP_DDHHMMSS := $(shell date +%d%H%M%S)
IMAGE_TAG ?= $(STAMP_DDHHMMSS)-dev

ifeq ($(PHASE), "release")
  # CONTAINER_REPO uses docker, the default
  CONTAINER_ORG = soloio
  IMAGE_TAG = $(VERSION)
else ifeq ($(PHASE), "dev")
  # CONTAINER_REPO can override with env
  # CONTAINER_ORG can override with env
  # IMAGE_TAG can override with env
else ifeq ($(PHASE), "buildtest")
  CONTAINER_REPO = $(GCR_REPO_PREFIX)
  CONTAINER_ORG = soloio
  IMAGE_TAG = $(LAST_COMMIT)-buildtest
  # TODO - delete these images from the repo after the test runs
  # consider adding $(shell date +%m%d%H%M%s) to the end of this tag if it helps to clean old builds
endif

ifeq ($(CONTAINER_REPO),)
  CONTAINER_REPO_ORG=$(CONTAINER_ORG)
else
  CONTAINER_REPO_ORG=$(CONTAINER_REPO)/$(CONTAINER_ORG)
endif

# For value debugging or preview
define MAKE_CONFIGURATION
Build state
 phase: $(PHASE)
Images configuration
 repo: $(CONTAINER_REPO)
 org: $(CONTAINER_ORG)
 tag: $(IMAGE_TAG)
 gcloud_project_id: $(GCLOUD_PROJECT_ID)
 full_spec: $(CONTAINER_REPO_ORG)
 sample: $(CONTAINER_REPO_ORG)/<container_name>:$(IMAGE_TAG)
endef
export MAKE_CONFIGURATION
.PHONY: print_configuration
print_configuration:
	echo "$$MAKE_CONFIGURATION"

#--- Specify project-specific constants and import project-specific build logic ---#
# import the targets that are common to many solo projects
FORMAT_DIRS = ./pkg/ ./cmd/ ./ci/
include make/common.makefile

SOURCES := $(shell find . -name "*.go" | grep -v test.go | grep -v '\.\#*')
LDFLAGS := "-X github.com/solo-io/$(SOLO_NAME)/pkg/version.Version=$(VERSION)"
GCFLAGS := all="-N -l"
include make/glooshot.makefile

include make/manifest.makefile

# these are phase-specific
ifeq ($(PHASE), "dev")
  include make/phase_dev.makefile
endif

.PHONY: docker
docker: glooshot-cli glooshot-operator glooshot-docker

.PHONY: docker-push
docker-push: glooshot-docker-push
	docker push $(CONTAINER_REPO_ORG)/$(GLOOSHOT_OPERATOR_NAME):$(IMAGE_TAG)

.PHONY: release
release: render-yaml docker-push
ifeq ($(PHASE), "release")
	go run ci/upload_github_release_assets.go
else
	echo "Cannot release in phase " $(PHASE)
endif
