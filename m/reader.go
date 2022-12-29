package m

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma/v2"
	log "github.com/sirupsen/logrus"
)

// Reader reads a file into an array of strings.
//
// It does the reading in the background, and it returns parts of the read data
// upon request.
//
// This package provides query methods for the struct, no peeking!!
type Reader struct {
	lines   []*Line
	name    *string
	lock    *sync.Mutex
	err     error
	_stderr io.Reader

	// Have we had our contents replaced using setText()?
	replaced bool

	done             chan bool
	highlightingDone chan bool // Used by tests
	moreLinesAdded   chan bool
}

// InputLines contains a number of lines from the reader, plus metadata
type InputLines struct {
	lines []*Line

	// One-based line number of the first line returned
	firstLineOneBased int

	// "monkey.txt: 1-23/45 51%"
	statusText string
}

// Shut down the filter (if any) after we're done reading the file.
func (reader *Reader) cleanupFilter(fromFilter *exec.Cmd) {
	// FIXME: Close the stream now that we're done reading it?

	if fromFilter == nil {
		reader.done <- true
		return
	}

	reader.lock.Lock()
	defer reader.lock.Unlock()

	// Give the filter a little time to go away
	timer := time.AfterFunc(2*time.Second, func() {
		// FIXME: Regarding error handling, maybe we should log all errors
		// except for "process doesn't exist"? If the process is not there
		// it likely means the process finished up by itself without our
		// help.
		_ = fromFilter.Process.Kill()
	})

	stderrText := ""
	if reader._stderr != nil {
		// Drain the reader's stderr into a string for possible inclusion in an error message
		// From: https://stackoverflow.com/a/9650373/473672
		if buffer, err := io.ReadAll(reader._stderr); err == nil {
			stderrText = strings.TrimSpace(string(buffer))
		} else {
			log.Warn("Draining filter stderr failed: ", err)
		}
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
}

// Count lines in the original file and preallocate space for them.  Good
// performance improvement:
//
// go test -benchmem -benchtime=10s -run='^$' -bench 'ReadLargeFile'
func (reader *Reader) preAllocLines(originalFileName string) {
	lineCount, err := countLines(originalFileName)
	if err != nil {
		log.Warn("Line counting failed: ", err)
		return
	}

	reader.lock.Lock()
	defer reader.lock.Unlock()

	if len(reader.lines) == 0 {
		// We had no lines since before, this is the expected happy path.
		reader.lines = make([]*Line, 0, lineCount)
		return
	}

	// There are already lines in here, this is unexpected.
	if reader.replaced {
		// Highlighting already done (because that's how reader.replaced gets
		// set to true)
		log.Debug("Highlighting was faster than line counting for a ",
			len(reader.lines), " lines file, this is unexpected")
	} else {
		// Highlighing not done, where the heck did those lines come from?
		log.Warn("Already had ", len(reader.lines),
			" lines by the time counting was done, and it's not highlighting")
	}
}

// This function will be update the Reader struct in the background.
func (reader *Reader) readStream(stream io.Reader, originalFileName *string, fromFilter *exec.Cmd) {
	defer reader.cleanupFilter(fromFilter)

	if originalFileName != nil {
		reader.preAllocLines(*originalFileName)
	}

	bufioReader := bufio.NewReader(stream)
	completeLine := make([]byte, 0)
	t0 := time.Now().UnixNano()
	for {
		keepReadingLine := true
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

				reader.lock.Lock()
				if reader.err == nil {
					// Store the error unless it overwrites one we already have
					reader.err = fmt.Errorf("error reading line from input stream: %w", err)
				}
				reader.lock.Unlock()
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

		newLineString := string(completeLine)
		newLine := NewLine(newLineString)

		reader.lock.Lock()
		if reader.replaced {
			// Somebody called setText(), never mind reading the rest of this stream
			reader.lock.Unlock()
			break
		}
		reader.lines = append(reader.lines, &newLine)
		reader.lock.Unlock()
		completeLine = completeLine[:0]

		// This is how to do a non-blocking write to a channel:
		// https://gobyexample.com/non-blocking-channel-operations
		select {
		case reader.moreLinesAdded <- true:
		default:
			// Default case required for the write to be non-blocking
		}
	}

	t1 := time.Now().UnixNano()
	dtNanos := t1 - t0
	log.Debug("Stream read in ", dtNanos/1_000_000, "ms")
}

// NewReaderFromStream creates a new stream reader
//
// The name can be an empty string ("").
//
// If non-empty, the name will be displayed by the pager in the bottom left
// corner to help the user keep track of what is being paged.
func NewReaderFromStream(name string, reader io.Reader) *Reader {
	mReader := newReaderFromStream(reader, nil, nil)
	mReader.highlightingDone <- true // No highlighting of streams = nothing left to do = Done!

	if len(name) > 0 {
		mReader.lock.Lock()
		mReader.name = &name
		mReader.lock.Unlock()
	}

	return mReader
}

// newReaderFromStream creates a new stream reader
//
// originalFileName is used for counting the lines in the file. nil for
// don't-know (streams) or not countable (compressed files). The line count is
// then used for pre-allocating the lines slice, which improves large file
// loading performance.
//
// If fromFilter is not nil this method will wait() for it, and effectively
// takes over ownership for it.
func newReaderFromStream(reader io.Reader, originalFileName *string, fromFilter *exec.Cmd) *Reader {
	returnMe := Reader{
		lock: new(sync.Mutex),
		done: make(chan bool, 1),
		// This needs to be size 1. If it would be 0, and we add more
		// lines while the pager is processing, the pager would miss
		// the lines added while it was processing.
		moreLinesAdded:   make(chan bool, 1),
		highlightingDone: make(chan bool, 1),
	}

	// FIXME: Make sure that if we panic somewhere inside of this goroutine,
	// the main program terminates and prints our panic stack trace.
	go returnMe.readStream(reader, originalFileName, fromFilter)

	return &returnMe
}

// NewReaderFromText creates a Reader from a block of text.
//
// First parameter is the name of this Reader. This name will be displayed by
// Moar in the bottom left corner of the screen.
//
// Calling _wait() on this Reader will always return immediately, no
// asynchronous ops will be performed.
func NewReaderFromText(name string, text string) *Reader {
	noExternalNewlines := strings.Trim(text, "\n")
	lines := []*Line{}
	if len(noExternalNewlines) > 0 {
		for _, lineString := range strings.Split(noExternalNewlines, "\n") {
			line := NewLine(lineString)
			lines = append(lines, &line)
		}
	}
	done := make(chan bool, 1)
	highlightingDone := make(chan bool, 1)
	done <- true
	returnMe := &Reader{
		name:             &name,
		lines:            lines,
		lock:             &sync.Mutex{},
		done:             done,
		highlightingDone: highlightingDone,
	}

	returnMe.highlightingDone <- true // No highlighting to do = nothing left = Done!

	return returnMe
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

	reader := newReaderFromStream(filterOut, nil, filter)
	reader.highlightingDone <- true // No highlighting to do == nothing left == Done!
	reader.lock.Lock()
	reader.name = &filename
	reader._stderr = filterErr
	reader.lock.Unlock()
	return reader, nil
}

func tryOpen(filename string) error {
	// Try opening the file
	tryMe, err := os.Open(filename)
	if err != nil {
		return err
	}

	// Try reading a byte
	buffer := make([]byte, 1)
	_, err = tryMe.Read(buffer)

	if err != nil && err.Error() == "EOF" {
		// Empty file, this is fine
		err = nil
	}

	closeErr := tryMe.Close()
	if err == nil && closeErr != nil {
		// Everything worked up until Close(), report the Close() error
		return closeErr
	}

	return err
}

// From: https://stackoverflow.com/a/52153000/473672
func countLines(filename string) (uint64, error) {
	const lineBreak = '\n'
	sliceWithSingleLineBreak := []byte{lineBreak}

	reader, err := os.Open(filename)
	if err != nil {
		return 0, err
	}
	defer func() {
		err := reader.Close()
		if err != nil {
			log.Warn("Error closing file after counting the lines: ", err)
		}
	}()

	var count uint64
	t0 := time.Now().UnixNano()
	buf := make([]byte, bufio.MaxScanTokenSize)
	lastReadEndsInNewline := true
	for {
		bufferSize, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return 0, err
		}

		if bufferSize > 0 {
			lastReadEndsInNewline = (buf[bufferSize-1] == lineBreak)
		}

		count += uint64(bytes.Count(buf[:bufferSize], sliceWithSingleLineBreak))
		if err == io.EOF {
			break
		}
	}

	if !lastReadEndsInNewline {
		// No trailing line feed, this needs special handling
		count++
	}

	t1 := time.Now().UnixNano()
	dtNanos := t1 - t0
	if count == 0 {
		log.Debug("Counted ", count, " lines in ", dtNanos/1_000_000, "ms")
	} else {
		log.Debug("Counted ", count, " lines in ", dtNanos/1_000_000, "ms at ", dtNanos/int64(count), "ns/line")
	}
	return count, nil
}

// NewReaderFromFilename creates a new file reader.
//
// The Reader will try to uncompress various compressed file format, and also
// apply highlighting to the file using Chroma:
// https://github.com/alecthomas/chroma
func NewReaderFromFilename(filename string, style chroma.Style, formatter chroma.Formatter) (*Reader, error) {
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

	stream, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	returnMe := newReaderFromStream(stream, &filename, nil)
	returnMe.lock.Lock()
	returnMe.name = &filename
	returnMe.lock.Unlock()

	go func() {
		defer func() {
			returnMe.highlightingDone <- true
		}()

		highlighted, err := highlight(filename, false, style, formatter)
		if err != nil {
			log.Warn("Highlighting failed: ", err)
			return
		}

		if highlighted == nil {
			// No highlighting would be done, never mind
			return
		}

		returnMe.setText(*highlighted)
	}()

	return returnMe, nil
}

// createStatusUnlocked() assumes that its caller is holding the lock
func (r *Reader) createStatusUnlocked(firstLineOneBased int, lastLineOneBased int) string {
	prefix := ""
	if r.name != nil {
		prefix = path.Base(*r.name) + ": "
	}

	if len(r.lines) == 0 {
		return prefix + "<empty>"
	}

	if len(r.lines) == 1 {
		return prefix + "1 line  100%"
	}

	percent := int(100 * float64(lastLineOneBased) / float64(len(r.lines)))

	return fmt.Sprintf("%s%s lines  %d%%",
		prefix,
		formatNumber(uint(len(r.lines))),
		percent)
}

// GetLineCount returns the number of lines available for viewing
func (r *Reader) GetLineCount() int {
	r.lock.Lock()
	defer r.lock.Unlock()

	return len(r.lines)
}

// GetLine gets a line. If the requested line number is out of bounds, nil is returned.
func (r *Reader) GetLine(lineNumberOneBased int) *Line {
	r.lock.Lock()
	defer r.lock.Unlock()

	if lineNumberOneBased < 1 {
		return nil
	}
	if lineNumberOneBased > len(r.lines) {
		return nil
	}
	return r.lines[lineNumberOneBased-1]
}

// GetLines gets the indicated lines from the input
func (r *Reader) GetLines(firstLineOneBased int, wantedLineCount int) *InputLines {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.getLinesUnlocked(firstLineOneBased, wantedLineCount)
}

func (r *Reader) getLinesUnlocked(firstLineOneBased int, wantedLineCount int) *InputLines {
	if firstLineOneBased < 1 {
		firstLineOneBased = 1
	}

	if len(r.lines) == 0 {
		return &InputLines{
			lines: nil,

			// The line number set here won't matter, we'll clip it anyway when we get it back
			firstLineOneBased: 0,

			statusText: r.createStatusUnlocked(0, 0),
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

		return r.getLinesUnlocked(firstLineOneBased, wantedLineCount)
	}

	return &InputLines{
		lines:             r.lines[firstLineZeroBased : lastLineZeroBased+1],
		firstLineOneBased: firstLineOneBased,
		statusText:        r.createStatusUnlocked(firstLineOneBased, lastLineZeroBased+1),
	}
}

// Replace reader contents with the given text and mark as done
func (reader *Reader) setText(text string) {
	lines := []*Line{}
	for _, lineString := range strings.Split(text, "\n") {
		line := NewLine(lineString)
		lines = append(lines, &line)
	}

	if len(lines) > 0 && strings.HasSuffix(text, "\n") {
		// Input ends with an empty line. This makes our line count be
		// off-by-one, fix that!
		lines = lines[0 : len(lines)-1]
	}

	reader.lock.Lock()
	reader.lines = lines
	reader.replaced = true
	reader.lock.Unlock()

	select {
	case reader.done <- true:
	default:
	}

	select {
	case reader.moreLinesAdded <- true:
	default:
	}
}
