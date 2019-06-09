#!/bin/bash

# Verify sending the output to a file

set -e -o pipefail

RESULT="$(mktemp)"
function cleanup {
  rm -rf "$RESULT"
}
trap cleanup EXIT

echo Test reading from redirected stdin, writing to redirected stdout...
./moar < moar.go > "$RESULT"
diff -u moar.go "$RESULT"

echo Test redirecting a file by name into file by redirecting stdout...
./moar moar.go > "$RESULT"
diff -u moar.go "$RESULT"

echo Test redirecting non-existing file by name into redirected stdout...
RESULT="$(mktemp)"
if ./moar does-not-exist > "$RESULT" ; then
    echo ERROR: Should have failed on non-existing input file name
    exit 1
fi
