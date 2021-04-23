#!/bin/bash

set -e -o pipefail

# If you want to bump this, check here for the most recent release version:
# https://github.com/dominikh/go-tools/releases/latest
STATICCHECK_VERSION=2020.2.3
STATICCHECK="$(go env GOPATH)/bin/staticcheck"

# If you want to bump this, check here for the most recent release version:
# https://github.com/kisielk/errcheck/releases/latest
ERRCHECK_VERSION=v1.6.0
ERRCHECK="$(go env GOPATH)/bin/errcheck"

# Install our linters
echo Installing linters...
if [ "$0" -nt "$STATICCHECK" ] ; then
  go install "honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}"
fi
if [ "$0" -nt "$ERRCHECK" ] ; then
  go install "github.com/kisielk/errcheck@${ERRCHECK_VERSION}"
fi

# Test that we only pass tcell.Color constants to these methods, not numbers
grep -En 'Foreground\([1-9]' ./*.go ./*/*.go && exit 1
grep -En 'Background\([1-9]' ./*.go ./*/*.go && exit 1

# Compile test first
./build.sh

# Linting
echo "Checking code formatting..."
MISFORMATTED="$(gofmt -l -s .)"
if [ -n "$MISFORMATTED" ]; then
  echo >&2 "==="
  echo >&2 "ERROR: The following files are not formatted, run './build.sh', './test.sh' or 'gofmt -s -w .' to fix:"
  echo >&2 "$MISFORMATTED"
  echo >&2 "==="
  exit 1
fi

# "go vet" catches fmt-placeholders-vs-args problems (and others)
echo "Running go vet..."
if ! go vet . ./twin ./m ; then
  if [ -n "${CI}" ]; then
    echo >&2 "==="
    echo >&2 "=== Please run './test.sh' before pushing to see the above issues locally rather than in CI"
    echo >&2 "==="
  fi
  exit 1
fi

# Docs: https://staticcheck.io/docs/
echo "Running staticcheck..."
if ! "$STATICCHECK" -f stylish . ./... ; then
  if [ -n "${CI}" ]; then
    echo >&2 "==="
    echo >&2 "=== Please run './test.sh' before pushing to see the above issues locally rather than in CI"
    echo >&2 "==="
  fi
  exit 1
fi

# Checks for unused error return values: https://github.com/kisielk/errcheck
echo "Running errcheck to check for unused error return values..."
if ! "$ERRCHECK" . ./... ; then
  if [ -n "${CI}" ]; then
    echo >&2 "==="
    echo >&2 "=== Please run './test.sh' before pushing to see the above issues locally rather than in CI"
    echo >&2 "==="
  fi
  exit 1
fi

# Unit tests
echo "Running unit tests..."
go test -timeout 20s ./...

# Ensure we can cross compile
# NOTE: Make sure this list matches the one in release.sh
GOOS=linux GOARCH=386 ./build.sh
GOOS=darwin GOARCH=amd64 ./build.sh

# Verify sending the output to a file
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
if ./moar does-not-exist >& /dev/null ; then
    echo ERROR: Should have failed on non-existing input file name
    exit 1
fi

echo Test --version...
./moar --version > /dev/null  # Should exit with code 0
diff -u <(./moar --version) <(git describe --tags --dirty --always)

# FIXME: On unknown command line options, test that help text goes to stderr

echo
echo "All tests passed!"
