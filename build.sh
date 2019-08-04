#!/bin/bash

VERSION="$(git describe --tags --dirty)"

BINARY="moar"
if [ -n "$GOOS$GOARCH" ] ; then
    BINARY="$BINARY-$VERSION-$GOOS-$GOARCH"
fi

# Linker flags trick below from here:
# https://www.reddit.com/r/golang/comments/4cpi2y/question_where_to_keep_the_version_number_of_a_go/d1kbap7?utm_source=share&utm_medium=web2x

# This line must be last in the script so that its return code
# propagates properly to its caller
go build -ldflags="-X main.versionString=$VERSION" -o "$BINARY"
