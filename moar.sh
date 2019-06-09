#!/bin/bash

set -e -o pipefail

rm -f moar

go test 1>&2

go build 1>&2

./moar "$@"
