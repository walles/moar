#!/bin/bash

# Verify the PAGER= suggestion is correct in "moar --help" output.
#
# Ref: https://github.com/walles/moar/issues/88

set -e -o pipefail

MOAR="$(realpath "$1")"
if ! [ -x "$MOAR" ]; then
    echo ERROR: Not executable: "$MOAR"
    exit 1
fi

echo Testing PAGER= suggestion in moar --help output...

WORKDIR="$(mktemp -d -t moar-path-help-test.XXXXXXXX)"

# Put a symlink to $MOAR first in the $PATH
ln -s "$MOAR" "$WORKDIR/moar"
echo "moar" >"$WORKDIR/expected"

# Extract suggested PAGER value from moar --help
PATH="$WORKDIR" PAGER="" moar --help | grep "export PAGER" | sed -E 's/.*=//' >"$WORKDIR/actual"

# Ensure it matches the symlink we have in $PATH
cd "$WORKDIR"
diff -u actual expected
