package m

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// Reader reads a file into an array of strings.
//
// It does the reading in the background, and it returns parts of the read data
// upon request.
//
// This package provides query methods for the struct, no peeking!!
type Reader struct {
	lines   []string
	name    *string
	lock    *sync.Mutex
	err     error
	_stderr io.Reader

	done           chan bool
	moreLinesAdded chan bool
}

// Lines contains a number of lines from the reader, plus metadata
type Lines struct {
	lines []string

	// One-based line number of the first line returned
	firstLineOneBased int

	// "monkey.txt: 1-23/45 51%"
	statusText string
}

func readStream(stream io.Reader, reader *Reader, fromFilter *exec.Cmd) {
	// FIXME: Close the stream when done reading it?

	defer func() {
		reader.lock.Lock()
		defer reader.lock.Unlock()

		if fromFilter == nil {
			reader.done <- true
			return
		}

		// Give the filter a little time to go away
		timer := time.AfterFunc(2*time.Second, func() {
			fromFilter.Process.Kill()
		})

		stderrText := ""
		if reader._stderr != nil {
			// Drain the reader's stderr into a string for possible inclusion in an error message
			buffer := new(bytes.Buffer)
			buffer.ReadFrom(reader._stderr)
			stderrText = strings.TrimSpace(buffer.String())
		}

		err := fromFilter.Wait()
		timer.Stop()

		// Don't overwrite any existing problem report
		if reader.err == nil {
			reader.err = err
			if err != nil && stderrText != "" {
				reader.err = fmt.Errorf("%s: %w", stderrText, err)
			}
		}

		// FIXME: Report any filter printouts to stderr to the user

		// Must send non-blocking since the channel has no buffer and sometimes no reader
		select {
		case reader.done <- true:
		default:
			// Empty default statement required for the write to be non-blocking,
			// without this the write blocks and just hangs. Then we never get to
			// the deferred reader.lock.Unlock() (see above), and the pager hangs
			// when trying to take the lock for getting more lines.
		}
	}()

	bufioReader := bufio.NewReader(stream)
	for {
		keepReadingLine := true
		var completeLine []byte
		eof := false

		var lineBytes []byte
		var err error
		for keepReadingLine {
			lineBytes, keepReadingLine, err = bufioReader.ReadLine()
			if err != nil {
				if err == io.EOF {
					eof = true
					break
				}

				if reader.err == nil {
					// Store the error unless it overwrites one we already have
					reader.lock.Lock()
					reader.err = fmt.Errorf("Error reading line from input stream: %w", err)
					reader.lock.Unlock()
				}
				break
			}

			completeLine = append(completeLine, lineBytes...)
		}

		if eof {
			break
		}

		if reader.err != nil {
			break
		}

		reader.lock.Lock()
		reader.lines = append(reader.lines, string(completeLine))
		reader.lock.Unlock()

		// This is how to do a non-blocking write to a channel:
		// https://gobyexample.com/non-blocking-channel-operations
		select {
		case reader.moreLinesAdded <- true:
		default:
			// Default case required for the write to be non-blocking
		}
	}
}

// NewReaderFromStream creates a new stream reader
//
// The name can be an empty string ("").
//
// If non-empty, the name will be displayed by the pager in the bottom left
// corner to help the user keep track of what is being paged.
func NewReaderFromStream(name string, reader io.Reader) *Reader {
	mReader := newReaderFromStream(reader, nil)

	if len(name) > 0 {
		mReader.lock.Lock()
		mReader.name = &name
		mReader.lock.Unlock()
	}

	return mReader
}

// newReaderFromStream creates a new stream reader
//
// If fromFilter is not nil this method will wait() for it,
// and effectively takes over ownership for it.
func newReaderFromStream(reader io.Reader, fromFilter *exec.Cmd) *Reader {
	var lines []string
	var lock = &sync.Mutex{}
	done := make(chan bool, 1)

	// This needs to be size 1. If it would be 0, and we add more lines while the
	// pager is processing, the pager would miss the lines added while it was
	// processing.
	moreLinesAdded := make(chan bool, 1)

	returnMe := Reader{
		lines:          lines,
		lock:           lock,
		done:           done,
		moreLinesAdded: moreLinesAdded,
	}

	// FIXME: Make sure that if we panic somewhere inside of this goroutine,
	// the main program terminates and prints our panic stack trace.
	go readStream(reader, &returnMe, fromFilter)

	return &returnMe
}

// NewReaderFromText creates a Reader from a block of text.
//
// First parameter is the name of this Reader. This name will be displayed by
// Moar in the bottom left corner of the screen.
func NewReaderFromText(name string, text string) *Reader {
	noExternalNewlines := strings.Trim(text, "\n")
	done := make(chan bool, 1)
	done <- true
	return &Reader{
		name:  &name,
		lines: strings.Split(noExternalNewlines, "\n"),
		lock:  &sync.Mutex{},
		done:  done,
	}
}

// Wait for reader to finish reading. Used by tests.
func (r *Reader) _Wait() error {
	// Wait for our goroutine to finish
	<-r.done

	r.lock.Lock()
	defer r.lock.Unlock()
	return r.err
}

// newReaderFromCommand creates a new reader by running a file through a filter
func newReaderFromCommand(filename string, filterCommand ...string) (*Reader, error) {
	filterWithFilename := append(filterCommand, filename)
	filter := exec.Command(filterWithFilename[0], filterWithFilename[1:]...)

	filterOut, err := filter.StdoutPipe()
	if err != nil {
		return nil, err
	}

	filterErr, err := filter.StderrPipe()
	if err != nil {
		// The error stream is only used in case of failures, and having it
		// nil is fine, so just log this and move along.
		log.Warnf("Stderr not available from %s: %s", filterCommand[0], err.Error())
	}

	err = filter.Start()
	if err != nil {
		return nil, err
	}

	reader := newReaderFromStream(filterOut, filter)
	reader.lock.Lock()
	reader.name = &filename
	reader._stderr = filterErr
	reader.lock.Unlock()
	return reader, nil
}

func canHighlight(filename string) bool {
	extension := filepath.Ext(filename)
	if len(extension) <= 1 {
		// No extension or a single "."
		return false
	}

	if extension == ".txt" {
		// Highlighting text files won't be of much use.
		//
		// Also: https://github.com/walles/moar/issues/29
		return false
	}

	// Remove leading dot from the extension
	extension = extension[1:]

	// Check file extension vs "highlight --list-scripts=langs" before calling
	// highlight, otherwise files with unsupported extensions (like .log) get
	// messed upp.
	highlight := exec.Command("highlight", "--list-scripts=langs")
	outBytes, err := highlight.CombinedOutput()
	if err != nil {
		return false
	}

	extensionMatcher := regexp.MustCompile("[^() ]+")

	outString := string(outBytes)
	outLines := strings.Split(outString, "\n")
	for _, line := range outLines {
		parts := strings.Split(line, ": ")
		if len(parts) < 2 {
			continue
		}

		// Pick out all extensions from this line
		for _, supportedExtension := range extensionMatcher.FindAllString(parts[1], -1) {
			if extension == supportedExtension {
				return true
			}
		}
	}

	// No match
	return false
}

func tryOpen(filename string) error {
	// Try opening the file
	tryMe, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer tryMe.Close()

	// Try reading a byte
	buffer := make([]byte, 1)
	_, err = tryMe.Read(buffer)

	if err != nil && err.Error() == "EOF" {
		// Empty file, this is fine
		return nil
	}

	return err
}

// NewReaderFromFilename creates a new file reader.
//
// The Reader will try to uncompress various compressed file format, and also
// apply highlighting to the file using highlight:
// http://www.andre-simon.de/doku/highlight/en/highlight.php
func NewReaderFromFilename(filename string) (*Reader, error) {
	fileError := tryOpen(filename)
	if fileError != nil {
		return nil, fileError
	}

	if strings.HasSuffix(filename, ".gz") {
		return newReaderFromCommand(filename, "gzip", "-d", "-c")
	}
	if strings.HasSuffix(filename, ".bz2") {
		return newReaderFromCommand(filename, "bzip2", "-d", "-c")
	}
	if strings.HasSuffix(filename, ".xz") {
		return newReaderFromCommand(filename, "xz", "-d", "-c")
	}

	// Highlight input file using highlight:
	// http://www.andre-simon.de/doku/highlight/en/highlight.php
	if canHighlight(filename) {
		highlighted, err := newReaderFromCommand(filename, "highlight", "--out-format=esc", "-i")
		if err == nil {
			return highlighted, err
		}
	}

	stream, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	reader := NewReaderFromStream(filename, stream)
	return reader, nil
}

// _CreateStatusUnlocked() assumes that its caller is holding the lock
func (r *Reader) _CreateStatusUnlocked(firstLineOneBased int, lastLineOneBased int) string {
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

			statusText: r._CreateStatusUnlocked(0, 0),
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
		statusText:        r._CreateStatusUnlocked(firstLineOneBased, lastLineZeroBased+1),
	}
}
