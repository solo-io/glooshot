

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
