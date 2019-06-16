#!/bin/bash

set -e -o pipefail

rm -f moar

go build 1>&2

./moar "$@"
