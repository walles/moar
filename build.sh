#!/bin/bash

VERSION="$(git describe --dirty)"

BINARY="moar"
if [ -n "$GOOS$GOARCH" ] ; then
    BINARY="releases/$BINARY-$VERSION-$GOOS-$GOARCH"
fi

# Linker flags version number trick below from here:
# https://www.reddit.com/r/golang/comments/4cpi2y/question_where_to_keep_the_version_number_of_a_go/d1kbap7?utm_source=share&utm_medium=web2x

# Linker flags -s and -w strips debug data, but keeps whatever is needed for
# proper panic backtraces, this makes binaries smaller:
# https://boyter.org/posts/trimming-golang-binary-fat/

# This line must be last in the script so that its return code
# propagates properly to its caller
go build -ldflags="-s -w -X main.versionString=$VERSION" -o "$BINARY"
