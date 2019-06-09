#!/bin/bash

set -e -o pipefail

rm -f moar

go test

go build

./moar "$@"
