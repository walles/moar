#!/bin/bash

# Verify sending the output to a file

set -e -o pipefail

echo Test redirecting a file by name into file by redirecting stdout...
RESULT="$(mktemp)"
./moar moar.go > "$RESULT"
diff -u moar.go "$RESULT"
rm "$RESULT"

echo Test reading from redirected stdin, writing to redirected stdout...
RESULT="$(mktemp)"
./moar < moar.go > "$RESULT"
diff -u moar.go "$RESULT"
rm "$RESULT"
