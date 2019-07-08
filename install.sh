#!/bin/bash

set -e -o pipefail

./test.sh

echo

echo 'Installing into /usr/local/bin...'
cp moar /usr/local/bin/moar

echo
echo 'Installed, try "moar moar.go" to see moar in action!'
