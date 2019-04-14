SOLO_NAME := glooshot
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

# default to the docker repo https://docs.docker.com/engine/reference/commandline/tag/
CONTAINER_REPO ?= registry-1.docker.io

LDFLAGS := "-X github.com/solo-io/$(SOLO_NAME)/pkg/version.Version=$(VERSION)"
GCFLAGS := all="-N -l"

# Passed by cloudbuild
GCLOUD_PROJECT_ID := $(GCLOUD_PROJECT_ID)
BUILD_ID := $(BUILD_ID)

TEST_IMAGE_TAG := test-$(BUILD_ID)
TEST_ASSET_DIR := $(ROOTDIR)/_test
GCR_REPO_PREFIX := gcr.io/$(GCLOUD_PROJECT_ID)

include make/common.makefile

include make/update.makefile

include make/generated_code.makefile

include make/manifest.makefile

include make/glooshot.makefile

include make/release.makefile

include make/containers.makefile
