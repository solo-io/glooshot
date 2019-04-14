#!/bin/bash
set -ex
go get -u golang.org/x/tools/cmd/goimports
go get -u github.com/gogo/protobuf/gogoproto
go get -u github.com/gogo/protobuf/protoc-gen-gogo
go get -u github.com/envoyproxy/protoc-gen-validate
go get -u github.com/paulvollmer/2gobytes
