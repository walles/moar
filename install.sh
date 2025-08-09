#!/bin/bash

set -e -o pipefail

./test.sh

echo

echo 'Installing into /usr/local/bin...'
cp moor /usr/local/bin/moor

echo
echo 'Installed, try "moor moor.go" to see moor in action!'
