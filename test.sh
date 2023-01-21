#!/bin/bash

set -e -o pipefail

# Test that we only pass twin colors to these methods, not numbers
grep -En 'Foreground\([1-9]' ./*.go ./*/*.go && exit 1
grep -En 'Background\([1-9]' ./*.go ./*/*.go && exit 1

# Compile test first
echo Building sources...
./build.sh

# Linting
echo 'Linting, repro any errors locally using "golangci-lint run"...'
echo '  Linting without tests...'
golangci-lint run --tests=false
echo '  Linting with tests...'
golangci-lint run --tests=true

# Unit tests
echo "Running unit tests..."
go test -timeout 20s ./...

# Ensure we can cross compile
echo "Testing cross compilation..."
source arches.sh
arches

# Verify sending the output to a file
RESULT="$(mktemp)"
function cleanup {
  rm -rf "${RESULT}"
}
trap cleanup EXIT

echo Test reading from redirected stdin, writing to redirected stdout...
./moar <moar.go >"${RESULT}"
diff -u moar.go "${RESULT}"

echo Test redirecting a file by name into file by redirecting stdout...
./moar moar.go >"${RESULT}"
diff -u moar.go "${RESULT}"

echo Test redirecting non-existing file by name into redirected stdout...
if ./moar does-not-exist >&/dev/null; then
  echo ERROR: Should have failed on non-existing input file name
  exit 1
fi

echo Test --version...
./moar --version >/dev/null # Should exit with code 0
diff -u <(./moar --version) <(git describe --tags --dirty --always)

echo Test that the man page and --help document the same set of options...
MAN_OPTIONS="$(grep -E '^\\fB' moar.1 | cut -d\\ -f4- | sed 's/fR.*//' | sed 's/\\//g')"
MOAR_OPTIONS="$(./moar --help | grep -E '^  -' | cut -d' ' -f3 | grep -v -- -version)"
diff -u <(echo "${MAN_OPTIONS}") <(echo "${MOAR_OPTIONS}")

# FIXME: On unknown command line options, test that help text goes to stderr

./scripts/test-path-help.sh "$(realpath ./moar)"

echo
echo "All tests passed!"
