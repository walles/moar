#!/bin/bash

set -e -o pipefail

# Unit tests first
go test github.com/walles/moar/m

# Ensure we can cross compile
GOOS=linux GOARCH=386 ./build.sh
GOOS=darwin GOARCH=amd64 ./build.sh

# Make sure we have a runnable binary for the current platform when done
./build.sh

# Verify sending the output to a file
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

echo Test --version...
./moar --version > /dev/null  # Should exit with code 0
diff -u <(./moar --version) <(git describe --tags --dirty)

echo
echo "All tests passed!"
