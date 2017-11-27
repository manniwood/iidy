#!/bin/bash

set -e
set -u
set -o pipefail

protoc -I=. --go_out=plugins=grpc:. ./iidy.proto

