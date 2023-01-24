# Not intended to be run directly.
echo "  Linux i386..."
GOOS=linux GOARCH=386 ./build.sh
echo "  Linux amd64..."
GOOS=linux GOARCH=amd64 ./build.sh
echo "  Linux arm..."
GOOS=linux GOARCH=arm64 ./build.sh
echo "  Linux arm64..."
GOOS=linux GOARCH=arm64 ./build.sh
echo "  Windows 386..."
GOOS=windows GOARCH=386 ./build.sh
echo "  Windows amd64..."
GOOS=windows GOARCH=amd64 ./build.sh
