#!/bin/bash

set -e -o pipefail

# Test all permutations of the following:
# - Read from file vs from stdin
# - 'q' to quit vs 'v' to launch an editor
# - Terminal editor (nano) or GUI editor (code -w)

read -r -p "Press enter to start testing, then q to exit the pager"
clear
# With --trace we always get a non-zero exit code
./moor.sh --trace moor.sh || true

echo
read -r -p "Press enter to continue, then q to exit the pager"
clear
./moor.sh --trace <moor.sh || true

echo
read -r -p "Press enter to continue, then v to launch a terminal editor, then exit that"
clear
EDITOR=nano ./moor.sh --trace moor.sh || true

echo
read -r -p "Press enter to continue, then v to launch a terminal editor, then exit that"
clear
EDITOR=nano ./moor.sh --trace <moor.sh || true

echo
read -r -p "Press enter to continue, then v to launch a GUI editor, then exit that"
clear
EDITOR="code -w" ./moor.sh --trace moor.sh || true

echo
read -r -p "Press enter to continue, then v to launch a GUI editor, then exit that"
clear
EDITOR="code -w" ./moor.sh --trace <moor.sh || true
