#----------------------------------------------------------------------------------
# Repo setup
#----------------------------------------------------------------------------------
# https://www.viget.com/articles/two-ways-to-share-git-hooks-with-your-team/
.PHONY: init
init:
	git config core.hooksPath .githooks

.PHONY: pin-repos
pin-repos:
	go run make/pin_repos.go


.PHONY: update-deps
update-deps:
	make/update_deps.sh

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
