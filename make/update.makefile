

# note that the file path is resolved from the directory into which this file is make-included
.PHONY: update-deps
update-deps:
	make/update_deps.sh
