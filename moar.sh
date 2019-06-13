#!/bin/bash

set -e -o pipefail

rm -f moar

go test github.com/walles/moar/m 1>&2

go build 1>&2

./moar "$@"
