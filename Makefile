SOLO_NAME := glooshot
ROOTDIR := $(shell pwd)
OUTPUT_DIR ?= $(ROOTDIR)/_output

PHASE:="dev"
RELEASE := "false"
ifdef $(TAGGED_VERSION)
	RELEASE = "true"
  PHASE = "release"
else
  TAGGED_VERSION := vdev
endif
VERSION ?= $(shell echo $(TAGGED_VERSION) | cut -c 2-)
LAST_COMMIT = $(shell git rev-parse HEAD | cut -c 1-6)

CONTAINER_ORG ?= soloio
# CONTAINER_REPO := $(CONTAINER_REPO) # defaults to docker
# the docker documentation states the implied repo url is: registry-1.docker.io
# https://docs.docker.com/engine/reference/commandline/tag/
# just in case, we will let the docker tool provide that
GCLOUD_PROJECT_ID := $(GCLOUD_PROJECT_ID) # Passed by cloudbuild
BUILD_ID := $(BUILD_ID) # Passed by cloudbuild
GCR_REPO_PREFIX := gcr.io/$(GCLOUD_PROJECT_ID)

# default to DDHHMMSS-dev
IMAGE_TAG ?= $(shell date +%d%H%M%S)-dev

ifeq ($(PHASE), "release")
  CONTAINER_REPO = "" # to use docker, the default
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
 full_spec: $(CONTAINER_REPO_ORG)
 sample: $(CONTAINER_REPO_ORG)/<container_name>:$(IMAGE_TAG)
endef
export MAKE_CONFIGURATION
.PHONY: print_configuration
print_configuration:
	echo "$$MAKE_CONFIGURATION"






# import the targets that are common to many solo projects
FORMAT_DIRS = ./pkg/ ./cmd/ ./ci/
include make/common.makefile

SOURCES := $(shell find . -name "*.go" | grep -v test.go | grep -v '\.\#*')
LDFLAGS := "-X github.com/solo-io/$(SOLO_NAME)/pkg/version.Version=$(VERSION)"
GCFLAGS := all="-N -l"
TEST_IMAGE_TAG := test-$(BUILD_ID)
TEST_ASSET_DIR := $(ROOTDIR)/_test
include make/glooshot.makefile

include make/manifest.makefile

# these are phase-specific
ifeq ($(PHASE), "release")
  include make/phase_release.makefile
endif
ifeq ($(PHASE), "dev")
  include make/phase_dev.makefile
endif
