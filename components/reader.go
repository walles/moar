package reader

import (
	"io"
	"os"
	"bufio"
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
}

func NewReader(r io.Reader) (*_Reader, error) {
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
