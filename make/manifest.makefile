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
	go run install/helm/$(SOLO_NAME)/generate/cmd/generate.go $(VERSION)

.PHONY: update-helm-chart
update-helm-chart:
ifeq ($(PHASE),"release")
	mkdir -p $(HELM_SYNC_DIR)/charts
	helm package --destination $(HELM_SYNC_DIR)/charts $(HELM_DIR)
	helm repo index $(HELM_SYNC_DIR)
endif

HELMFLAGS := --namespace $(INSTALL_NAMESPACE) --set namespace.create=true

install/$(SOLO_NAME).yaml: prepare-helm
	helm template install/helm/$(SOLO_NAME) $(HELMFLAGS) > $@

.PHONY: render-yaml
render-yaml: install/$(SOLO_NAME).yaml
