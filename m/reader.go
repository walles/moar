package m

import (
	"bufio"
	"io"
	"os"
	"os/exec"
)

// Reader reads a file into an array of strings.
//
// When this thing grows up it's going to do the reading in the
// background, and it will return parts of the read data upon
// request.
//
// This package should provide query methods for the struct, no peeking!!
//
// FIXME: Make the reader read in the background, independently of what the pager is showing
type Reader struct {
	lines []string
	name  string
}

// Lines contains a number of lines from the reader, plus metadata
type Lines struct {
	lines []string

	// One-based number of the first of the lines
	firstLineOneBased int
}

// NewReaderFromStream creates a new stream reader
func NewReaderFromStream(reader io.Reader) (*Reader, error) {
	// FIXME: Close the stream when done reading it?
	scanner := bufio.NewScanner(reader)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return &Reader{
		lines: lines,
	}, nil
}

// Highlight input file using highlight:
// http://www.andre-simon.de/doku/highlight/en/highlight.php
func _Highlight(filename string) (io.Reader, error) {
	// FIXME: Check file extension vs "highlight --list-scripts=langs" before
	// calling highlight, otherwise binary files like /bin/ls get messed up.

	highlight := exec.Command("highlight", "--out-format=esc", "-i", filename)

	highlightOut, err := highlight.StdoutPipe()
	if err != nil {
		// FIXME: Print user warning when done if highlight isn't installed
		return nil, err
	}

	err = highlight.Start()
	if err != nil {
		return nil, err
	}

	return highlightOut, nil
}

// NewReaderFromFilename creates a new file reader
func NewReaderFromFilename(filename string) (*Reader, error) {
	stream, err := _Highlight(filename)
	if err != nil {
		// Can't highlight, try without
		stream, err = os.Open(filename)
		if err != nil {
			return nil, err
		}
	}

	reader, err := NewReaderFromStream(stream)
	if err != nil {
		return nil, err
	}

	reader.name = filename
	return reader, err
}

// GetLines gets the indicated lines from the input
func (r *Reader) GetLines(firstLineOneBased int, wantedLineCount int) *Lines {
	if firstLineOneBased < 1 {
		firstLineOneBased = 1
	}

	if len(r.lines) == 0 {
		return &Lines{
			lines: r.lines,

			// FIXME: What line number should we set here?
			firstLineOneBased: firstLineOneBased,
		}
	}

	firstLineZeroBased := firstLineOneBased - 1
	lastLineZeroBased := firstLineZeroBased + wantedLineCount - 1

	if lastLineZeroBased >= len(r.lines) {
		lastLineZeroBased = len(r.lines) - 1
	}

	// Prevent reading past the end of the available lines
	actualLineCount := lastLineZeroBased - firstLineZeroBased + 1
	if actualLineCount < wantedLineCount && firstLineOneBased > 1 {
		overshoot := wantedLineCount - actualLineCount
		firstLineOneBased -= overshoot
		if firstLineOneBased < 1 {
			firstLineOneBased = 1
		}

		return r.GetLines(firstLineOneBased, wantedLineCount)
	}

	return &Lines{
		lines:             r.lines[firstLineZeroBased : lastLineZeroBased+1],
		firstLineOneBased: firstLineOneBased,
	}
}