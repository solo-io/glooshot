#----------------------------------------------------------------------------------
# Generated Code and Docs
#----------------------------------------------------------------------------------

.PHONY: generated-code
generated-code: $(OUTPUT_DIR)/.generated-code

SUBDIRS:=pkg cmd ci
$(OUTPUT_DIR)/.generated-code:
	go run generate.go
	gofmt -w $(SUBDIRS)
	goimports -w $(SUBDIRS)
	mkdir -p $(OUTPUT_DIR)
	touch $@
