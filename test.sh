#!/bin/bash

set -e

# Ensure we can cross compile
GOOS=linux GOARCH=386 go build
GOOS=darwin GOARCH=amd64 go build

# Unit tests first...
go test

# ... then integration.
go build
for TEST in integration-tests/*.sh ; do
    echo "Executing: $(basename "$TEST")..."
    "$TEST"
done
