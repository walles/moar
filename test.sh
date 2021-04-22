#!/bin/bash

set -e -o pipefail

# If you want to bump this, check here for the most recent release version:
# https://github.com/dominikh/go-tools/releases/latest
STATICCHECK_VERSION=2020.2.3

# If you want to bump this, check here for the most recent release version:
# https://github.com/kisielk/errcheck/releases/latest
ERRCHECK_VERSION=v1.6.0

# Install our linters
echo Installing linters...
go install "honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}"
go install "github.com/kisielk/errcheck@${ERRCHECK_VERSION}"

if [ -n "${CI}" ]; then
  # We want our linter versions listed in go.mod: https://github.com/walles/moar/pull/48
  #
  # Otherwise you'll get a change recorded every time you do './test.sh' locally, and we don't want that.
  MODIFIED="$(git status --porcelain go.mod go.sum)"
  if [ -n "${MODIFIED}" ]; then
    # The above "go get" invocation modified go.mod
    echo >&2 "==="
    git --no-pager diff
    echo >&2 "ERROR: go.mod modified by installing linters, run './test.sh' locally or this, commit your changes and try again:"

    # Must match the actual "go get" line higher up in this script
    echo >&2 '$' go get "honnef.co/go/tools/cmd/staticcheck@${STATICCHECK_VERSION}" "github.com/kisielk/errcheck@${ERRCHECK_VERSION}"

    echo >&2 "==="
    exit 1
  fi
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
if ! "$(go env GOPATH)/bin/staticcheck" -f stylish . ./... ; then
  if [ -n "${CI}" ]; then
    echo >&2 "==="
    echo >&2 "=== Please run './test.sh' before pushing to see the above issues locally rather than in CI"
    echo >&2 "==="
  fi
  exit 1
fi

# Checks for unused error return values: https://github.com/kisielk/errcheck
echo "Running errcheck to check for unused error return values..."
if ! "$(go env GOPATH)/bin/errcheck" . ./... ; then
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
