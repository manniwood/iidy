#!/bin/bash

set -e
set -u
set -o pipefail

protoc \
	-I=/usr/local/include \
	-I=. \
	-I=$GOPATH/src \
	-I=$GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis \
	--go_out=plugins=grpc:. \
	--grpc-gateway_out=logtostderr=true:. \
	./iidy.proto

