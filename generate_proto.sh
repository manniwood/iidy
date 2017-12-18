#!/bin/bash

set -e
set -u
set -o pipefail

export LD_LIBRARY_PATH=/usr/local/lib

protoc \
	-I=/usr/local/include \
	-I=. \
	-I=$GOPATH/src \
	-I=$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	--go_out=plugins=grpc:. \
	--grpc-gateway_out=logtostderr=true:. \
	./iidy.proto

