#----------------------------------------------------------------------------------
# glooshot
#----------------------------------------------------------------------------------

GLOOSHOT_DIR=cmd/glooshot
GLOOSHOT_SOURCES=$(shell find $(GLOOSHOT_DIR) -name "*.go" | grep -v test | grep -v generated.go)

$(OUTPUT_DIR)/$(SOLO_NAME)-linux-amd64: $(GLOOSHOT_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_DIR)/main.go

$(OUTPUT_DIR)/$(SOLO_NAME)-darwin: $(GLOOSHOT_SOURCES)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -ldflags=$(LDFLAGS) -gcflags=$(GCFLAGS) -o $@ $(GLOOSHOT_DIR)/main.go

.PHONY: glooshot
glooshot: $(OUTPUT_DIR)/$(SOLO_NAME)-linux-amd64 $(OUTPUT_DIR)/$(SOLO_NAME)-darwin

$(OUTPUT_DIR)/Dockerfile.glooshot: $(GLOOSHOT_DIR)/Dockerfile
	cp $< $@

.PHONY: glooshot-docker
glooshot-docker: $(OUTPUT_DIR)/glooshot-docker

$(OUTPUT_DIR)/glooshot-docker: $(OUTPUT_DIR)/$(SOLO_NAME)-linux-amd64 $(OUTPUT_DIR)/Dockerfile.glooshot
	docker build -t soloio/$(SOLO_NAME):$(VERSION) $(call get_test_tag_option,$(SOLO_NAME)) $(OUTPUT_DIR) -f $(OUTPUT_DIR)/Dockerfile.glooshot
	touch $@
