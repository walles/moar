#!/bin/bash

set -e -o pipefail

# Ensure we can cross compile
GOOS=linux GOARCH=amd64 go build
GOOS=linux GOARCH=386 go build
GOOS=darwin GOARCH=amd64 go build

go test

# Make sure we're built for the current platform
go build

./moar "$@"
