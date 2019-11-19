#!/bin/bash

set -e -o pipefail

echo "Running tests before making the release..."
./test.sh

# Bail if we're on a dirty version
if [ -n "$(git diff --stat)" ]; then
  echo "ERROR: Please commit all changes before doing a release"
  echo
  git status

  exit 1
fi

# List existing version numbers...
echo
echo "Previous version numbers:"
git tag | cat

# ... and ask for a new version number.
echo
echo "Please provide a version number for the new release:"
read -r VERSION

# List changes since last release as inspiration...
LAST_VERSION="$(git describe --abbrev=0)"
echo

# FIXME: Make this part of the editable tagging message
echo "Changes since last release:"
git log --first-parent --pretty="format:* %s" "$LAST_VERSION"..HEAD | sed 's/ diff.*//'
echo
echo

# Make an annotated tag for this release
git tag --annotate "$VERSION"

# NOTE: To get the version number right, these builds must
# be done after the above tagging.
#
# NOTE: Make sure this list matches the one in test.sh
GOOS=linux GOARCH=386 ./build.sh
GOOS=darwin GOARCH=amd64 ./build.sh

# Push the newly built release tag
git push --tags

# FIXME: Instead of asking the user to upload the binaries,
# upload them for the user.
echo
echo "Please upload the following binaries to <https://github.com/walles/moar/releases/tag/$VERSION>:"
file moar-"$VERSION"-*-*
