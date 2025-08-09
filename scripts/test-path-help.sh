#!/bin/bash

# Verify the PAGER= suggestion is correct in "moor --help" output.
#
# Ref: https://github.com/walles/moor/issues/88

set -e -o pipefail

MOOR="$(realpath "$1")"
if ! [ -x "$MOOR" ]; then
    echo ERROR: Not executable: "$MOOR"
    exit 1
fi

echo Testing PAGER= suggestion in moor --help output...

WORKDIR="$(mktemp -d -t moor-path-help-test.XXXXXXXX)"

# Put a symlink to $MOOR first in the $PATH
ln -s "$MOOR" "$WORKDIR/moor"
echo "moor" >"$WORKDIR/expected"

# Extract suggested PAGER value from moor --help
unset PAGER
PATH="$WORKDIR" PAGER="" MOOR="" moor --help | grep "PAGER" | grep -v "is empty" | sed -E 's/.*PAGER[= ]//' >"$WORKDIR/actual"

# Ensure it matches the symlink we have in $PATH
cd "$WORKDIR"
diff -u actual expected
