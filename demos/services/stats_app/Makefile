


IMAGE_TAG ?= dev
CONTAINER_REPO_ORG ?= docker.io/soloio

## These are the images created by this makefile
APP_SPEC := $(CONTAINER_REPO_ORG)/stats-app:$(IMAGE_TAG)

ROOTDIR := $(shell pwd)
OUTPUT_DIR := $(ROOTDIR)/_output

.PHONY: all
all: push-app

${OUTPUT_DIR}:
	mkdir ${OUTPUT_DIR}

.PHONY: compile
compile: ${OUTPUT_DIR}
	GOOS=linux GOARCH=amd64 go build -gcflags "-N -l" -o $(OUTPUT_DIR)/stats-app main.go

.PHONY: build-app
build-app: compile
	docker build -t $(APP_SPEC) -f Dockerfile .

.PHONY: push-app
push-app: build-app
	docker push $(APP_SPEC)



NLIST := localhost:8091,localhost:8092,localhost:8093
.PHONY: run-local-0
run-local-0:
	NEIGHBOR_INDEX=0 NEIGHBOR_SERVICE_LIST=$(NLIST) BIND_ADDRESS=localhost:8091 go run main.go
.PHONY: run-local-1
run-local-1:
	NEIGHBOR_INDEX=1 NEIGHBOR_SERVICE_LIST=$(NLIST) BIND_ADDRESS=localhost:8092 go run main.go
.PHONY: run-local-2
run-local-2:
	NEIGHBOR_INDEX=2 NEIGHBOR_SERVICE_LIST=$(NLIST) BIND_ADDRESS=localhost:8093 go run main.go


.PHONY: apply-yaml
apply-yaml:
	kubectl apply -f stats-app-all.yaml

.PHONY: delete-yaml
delete-yaml:
	kubectl delete -f stats-app-all.yaml

.PHONY: logs1
logs1:
	kubectl logs deploy/stats-app-1 stats-app -f


