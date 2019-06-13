#!/bin/bash

set -e -o pipefail

# Ensure we can cross compile
GOOS=linux GOARCH=386 go build
GOOS=darwin GOARCH=amd64 go build

# Unit tests first...
go test github.com/walles/moar/m

# ... then Verify sending the output to a file
go build

RESULT="$(mktemp)"
function cleanup {
  rm -rf "$RESULT"
}
trap cleanup EXIT

echo Running to-file redirection tests...

echo Test reading from redirected stdin, writing to redirected stdout...
./moar < moar.go > "$RESULT"
diff -u moar.go "$RESULT"

echo Test redirecting a file by name into file by redirecting stdout...
./moar moar.go > "$RESULT"
diff -u moar.go "$RESULT"

echo Test redirecting non-existing file by name into redirected stdout...
if ./moar does-not-exist >& /dev/null ; then
    echo ERROR: Should have failed on non-existing input file name
    exit 1
fi

echo
echo "All tests passed!"
