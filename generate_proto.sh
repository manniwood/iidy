#!/bin/bash

set -e
set -u
set -o pipefail

protoc -I=. --go_out=. ./iidy.proto

