#!/bin/bash

set -e -o pipefail

./test.sh

# FIXME: Bail if we're on a dirty version

# List existing version numbers...
echo
git tag | cat

# ... and ask for a new version number.
echo
echo "Please provide a version number for the new release:"
read -r VERSION

# FIXME: When asking for a release description, list
# changes since last release as inspiration

# Make an annotated tag for this release
git tag --annotated "$VERSION"

# NOTE: To get the version number right, these builds must
# be done after the above tagging.
#
# NOTE: Make sure this list matches the one in test.sh
GOOS=linux GOARCH=386 ./build.sh
GOOS=darwin GOARCH=amd64 ./build.sh

# Push the newly built release tag
git push --tags

# FIXME: Put the actual URL here, don't just refer to Github
#
# FIXME: Instead of asking the user to upload the binaries,
# upload them for the user.
echo
echo "Please upload the following binaries to the new release page on Github:"
find . -maxdepth 1 -name 'moar-*-*-*' -print0 | xargs -0 -n1 basename
