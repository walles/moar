#!/bin/bash

# Build pager and run it, this script should behave just
# like the binary.

set -e -o pipefail

MYDIR="$(cd "$(dirname "$0")"; pwd)"
cd "$MYDIR"

rm -f moar

./build.sh 1>&2

./moar "$@"
