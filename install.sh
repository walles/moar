#!/bin/bash

set -e -o pipefail

./test.sh

echo

set -x
cp moar /usr/local/bin/moar
