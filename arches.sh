#!/bin/bash
# Not intended to be run directly.
ARCHES_USE_LOGFILE=${ARCHES_USE_LOGFILE:=0}

_launch_arches() {
    coproc arches {
        echo "  Linux i386..."
        GOOS=linux GOARCH=386 ./build.sh
        echo "  Linux amd64..."
        GOOS=linux GOARCH=amd64 ./build.sh
        echo "  Linux arm..."
        GOOS=linux GOARCH=arm64 ./build.sh
        echo "  Linux arm64..."
        GOOS=linux GOARCH=arm64 ./build.sh
        echo "  macOS amd64..."
        GOOS=darwin GOARCH=amd64 ./build.sh
        echo "  macOS arm64..."
        GOOS=darwin GOARCH=arm64 ./build.sh
        echo "  Windows 386..."
        GOOS=windows GOARCH=386 ./build.sh
        echo "  Windows amd64..."
        GOOS=windows GOARCH=amd64 ./build.sh
    } 2>&1 1>&3
}

_wait_arches() {
    wait $arches_PID
    exec 3>&-
}

arches() {
    [ $ARCHES_USE_LOGFILE -eq 1 ] && exec 3<>arches.log || exec 3>&1
    tail -f /dev/fd/3 &
    local tail_PID=$!
    _launch_arches
    _wait_arches
    kill $tail_PID
    exec ${arches[1]}>&-
    [ $ARCHES_USE_LOGFILE -eq 1 ] && rm arches.log || :
    unset _launch_arches _wait_arches arches ARCHES_USE_LOGFILE
}

if [[ "$0" == "$BASH_SOURCE" ]]; then
    2>&1 echo Note: Not intended for direct use.
    arches
fi
