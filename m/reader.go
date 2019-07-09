package m

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
)

// Reader reads a file into an array of strings.
//
// When this thing grows up it's going to do the reading in the
// background, and it will return parts of the read data upon
// request.
//
// This package should provide query methods for the struct, no peeking!!
type Reader struct {
	lines []string
	name  *string
	lock  *sync.Mutex
	err   error
}

// Lines contains a number of lines from the reader, plus metadata
type Lines struct {
	lines []string

	// One-based line number of the first line returned
	firstLineOneBased int

	// "monkey.txt: 1-23/45 51%"
	statusText string
}

// NewReaderFromStream creates a new stream reader
//
// If fromFilter is not nil this method will wait() for it,
// and effectively takes over ownership for it.
func NewReaderFromStream(reader io.Reader, fromFilter *exec.Cmd) *Reader {
	// FIXME: Close the stream when done reading it?
	var lines []string
	var lock = &sync.Mutex{}

	returnMe := Reader{
		lines: lines,
		lock:  lock,
	}

	go func() {
		defer func() {
			returnMe.lock.Lock()
			defer returnMe.lock.Unlock()

			if fromFilter == nil {
				return
			}

			err := fromFilter.Wait()
			if returnMe.err == nil {
				returnMe.err = err
			}
		}()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			text := scanner.Text()
			returnMe.lock.Lock()
			returnMe.lines = append(returnMe.lines, text)
			returnMe.lock.Unlock()
		}

		if err := scanner.Err(); err != nil {
			returnMe.lock.Lock()
			returnMe.err = err
			returnMe.lock.Unlock()
			return
		}
	}()

	return &returnMe
}

// NewReaderFromCommand creates a new reader by running a file through a filter
func NewReaderFromCommand(filename string, filterCommand ...string) (*Reader, error) {
	filterWithFilename := append(filterCommand, filename)
	filter := exec.Command(filterWithFilename[0], filterWithFilename[1:]...)

	filterOut, err := filter.StdoutPipe()
	if err != nil {
		return nil, err
	}

	err = filter.Start()
	if err != nil {
		return nil, err
	}

	reader := NewReaderFromStream(filterOut, filter)
	reader.lock.Lock()
	reader.name = &filename
	reader.lock.Unlock()
	return reader, nil
}

// NewReaderFromFilename creates a new file reader
func NewReaderFromFilename(filename string) (*Reader, error) {
	if strings.HasSuffix(filename, ".gz") {
		return NewReaderFromCommand(filename, "gzip", "-d", "-c")
	}
	if strings.HasSuffix(filename, ".bz2") {
		return NewReaderFromCommand(filename, "bzip2", "-d", "-c")
	}
	if strings.HasSuffix(filename, ".xz") {
		return NewReaderFromCommand(filename, "xz", "-d", "-c")
	}

	// Highlight input file using highlight:
	// http://www.andre-simon.de/doku/highlight/en/highlight.php
	//
	// FIXME: Check file extension vs "highlight --list-scripts=langs" before
	// calling highlight, otherwise binary files like /bin/ls get messed up.
	highlighted, err := NewReaderFromCommand(filename, "highlight", "--out-format=esc", "-i")
	if err == nil {
		return highlighted, err
	}

	// FIXME: Warn user if highlight is not installed?

	stream, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	reader := NewReaderFromStream(stream, nil)
	reader.lock.Lock()
	reader.name = &filename
	reader.lock.Unlock()
	return reader, nil
}

// _CreateStatus() assumes that its caller is holding the lock
func (r *Reader) _CreateStatus(firstLineOneBased int, lastLineOneBased int) string {
	prefix := ""
	if r.name != nil {
		prefix = path.Base(*r.name) + ": "
	}

	if len(r.lines) == 0 {
		return prefix + "<empty>"
	}

	percent := int(math.Floor(100.0 * float64(lastLineOneBased) / float64(len(r.lines))))

	return fmt.Sprintf("%s%d-%d/%d %d%%",
		prefix,
		firstLineOneBased,
		lastLineOneBased,
		len(r.lines),
		percent)
}

// GetLineCount returns the number of lines available for viewing
func (r *Reader) GetLineCount() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	return len(r.lines)
}

// GetLine gets a line. If the requested line number is out of bounds, nil is returned.
func (r *Reader) GetLine(lineNumberOneBased int) *string {
	r.lock.Lock()
	defer r.lock.Unlock()

	if lineNumberOneBased < 1 {
		return nil
	}
	if lineNumberOneBased > len(r.lines) {
		return nil
	}
	return &r.lines[lineNumberOneBased-1]
}

// GetLines gets the indicated lines from the input
func (r *Reader) GetLines(firstLineOneBased int, wantedLineCount int) *Lines {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r._GetLinesUnlocked(firstLineOneBased, wantedLineCount)
}

func (r *Reader) _GetLinesUnlocked(firstLineOneBased int, wantedLineCount int) *Lines {
	if firstLineOneBased < 1 {
		firstLineOneBased = 1
	}

	if len(r.lines) == 0 {
		return &Lines{
			lines: nil,

			// The line number set here won't matter, we'll clip it anyway when we get it back
			firstLineOneBased: 0,

			statusText: r._CreateStatus(0, 0),
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

		return r._GetLinesUnlocked(firstLineOneBased, wantedLineCount)
	}

	return &Lines{
		lines:             r.lines[firstLineZeroBased : lastLineZeroBased+1],
		firstLineOneBased: firstLineOneBased,
		statusText:        r._CreateStatus(firstLineOneBased, lastLineZeroBased+1),
	}
}
