package m

import (
	"bufio"
	"io"
	"os"
)

// Reads a file into an array of strings.
//
// When this thing grows up it's going to do the reading in the
// background, and it will return parts of the read data upon
// request.
//
// This package should provide query methods for the struct, no peeking!!
type _Reader struct {
	lines []string
	name  string
}

// NewReaderFromStream creates a new stream reader
func NewReaderFromStream(r io.Reader) (*_Reader, error) {
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &_Reader{
		lines: lines,
	}, nil
}

// NewReaderFromFilename creates a new file reader
func NewReaderFromFilename(filename string) (*_Reader, error) {
	stream, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return NewReaderFromStream(stream)
}
