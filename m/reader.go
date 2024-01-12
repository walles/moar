package m

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/walles/moar/m/linenumbers"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	log "github.com/sirupsen/logrus"
)

// Files larger than this won't be highlighted
//
//revive:disable-next-line:var-naming
const MAX_HIGHLIGHT_SIZE int64 = 1024 * 1024

// Reader reads a file into an array of strings.
//
// It does the reading in the background, and it returns parts of the read data
// upon request.
//
// This package provides query methods for the struct, no peeking!!
type Reader struct {
	sync.Mutex

	lines []*Line
	name  *string
	err   error

	// Have we had our contents replaced using setText()?
	replaced bool

	done             *atomic.Bool
	highlightingDone *atomic.Bool

	// For telling the UI it should recheck the --quit-if-one-screen conditions.
	// Signalled when either highlighting is done or reading is done.
	maybeDone chan bool

	moreLinesAdded chan bool
}

// InputLines contains a number of lines from the reader, plus metadata
type InputLines struct {
	lines []*Line

	// One-based line number of the first line returned
	firstLine linenumbers.LineNumber

	// "monkey.txt: 1-23/45 51%"
	statusText string
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

	reader.Lock()
	defer reader.Unlock()

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
func (reader *Reader) readStream(stream io.Reader, originalFileName *string, onDone func()) {
	defer func() {
		reader.done.Store(true)
		select {
		case reader.maybeDone <- true:
		default:
		}
	}()

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

				reader.Lock()
				if reader.err == nil {
					// Store the error unless it overwrites one we already have
					reader.err = fmt.Errorf("error reading line from input stream: %w", err)
				}
				reader.Unlock()
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

		reader.Lock()
		if reader.replaced {
			// Somebody called setText(), never mind reading the rest of this stream
			reader.Unlock()
			break
		}
		reader.lines = append(reader.lines, &newLine)
		reader.Unlock()
		completeLine = completeLine[:0]

		// This is how to do a non-blocking write to a channel:
		// https://gobyexample.com/non-blocking-channel-operations
		select {
		case reader.moreLinesAdded <- true:
		default:
			// Default case required for the write to be non-blocking
		}
	}

	if onDone != nil {
		onDone()
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
func NewReaderFromStream(name string, reader io.Reader, style chroma.Style, formatter chroma.Formatter, lexer chroma.Lexer) *Reader {
	mReader := newReaderFromStream(reader, nil, style, formatter, lexer)

	if len(name) > 0 {
		mReader.Lock()
		mReader.name = &name
		mReader.Unlock()
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
// If lexer is not nil, the file will be highlighted after being fully read.
func newReaderFromStream(reader io.Reader, originalFileName *string, style chroma.Style, formatter chroma.Formatter, lexer chroma.Lexer) *Reader {
	done := atomic.Bool{}
	done.Store(false)
	highlightingDone := atomic.Bool{}
	highlightingDone.Store(false)
	returnMe := Reader{
		// This needs to be size 1. If it would be 0, and we add more
		// lines while the pager is processing, the pager would miss
		// the lines added while it was processing.
		moreLinesAdded:   make(chan bool, 1),
		maybeDone:        make(chan bool, 1),
		highlightingDone: &highlightingDone,
		done:             &done,
	}

	// FIXME: Make sure that if we panic somewhere inside of this goroutine,
	// the main program terminates and prints our panic stack trace.
	go returnMe.readStream(reader, originalFileName, func() {
		highlightFromMemory(&returnMe, style, formatter, lexer)
	})

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
	done := atomic.Bool{}
	done.Store(true)
	highlightingDone := atomic.Bool{}
	highlightingDone.Store(true) // No highlighting to do = nothing left = Done!
	returnMe := &Reader{
		lines:            lines,
		done:             &done,
		highlightingDone: &highlightingDone,
	}
	if name != "" {
		returnMe.name = &name
	}

	return returnMe
}

// Duplicate of moar/moar.go:tryOpen
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

	reader, err := ZOpen(filename)
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
		log.Debug("Counted ", count, " lines in 0ms")
	} else {
		log.Debug("Counted ", count, " lines in ", dtNanos/1_000_000, "ms at ", dtNanos/int64(count), "ns/line")
	}
	return count, nil
}

// NewReaderFromFilename creates a new file reader.
//
// If lexer is nil it will be determined from the input file name.
//
// The Reader will try to uncompress various compressed file format, and also
// apply highlighting to the file using Chroma:
// https://github.com/alecthomas/chroma
func NewReaderFromFilename(filename string, style chroma.Style, formatter chroma.Formatter, lexer chroma.Lexer) (*Reader, error) {
	fileError := tryOpen(filename)
	if fileError != nil {
		return nil, fileError
	}

	stream, err := ZOpen(filename)
	if err != nil {
		return nil, err
	}

	// Set lexer to nil in this call since we want to do our own highlighting in
	// parallel with the stream being read. See the call to
	// StartHighlightingFromFile() below.
	returnMe := newReaderFromStream(stream, &filename, chroma.Style{}, nil, nil)

	returnMe.Lock()
	returnMe.name = &filename
	returnMe.Unlock()

	startHighlightingFromFile(returnMe, filename, style, formatter, lexer)

	return returnMe, nil
}

func startHighlightingFromFile(reader *Reader, filename string, style chroma.Style, formatter chroma.Formatter, lexer chroma.Lexer) {
	reportDone := func() {
		reader.highlightingDone.Store(true)
		select {
		case reader.maybeDone <- true:
		default:
		}

		log.Trace("Highlighting done")
	}

	fileInfo, err := os.Stat(filename)
	if err != nil {
		log.Warn("Failed to stat file for highlighting: ", err)
		reportDone()
		return
	}
	if fileInfo.Size() > MAX_HIGHLIGHT_SIZE {
		log.Debug("File too large for highlighting: ", fileInfo.Size())
		reportDone()
		return
	}

	go func() {
		defer reportDone()

		fileBytes, err := os.ReadFile(filename)
		if err != nil {
			log.Warn("Failed to read file for highlighting: ", err)
			return
		}

		if lexer == nil {
			// Try auto detecting by filename
			lexer = lexers.Match(filename)
		}

		highlighted, err := highlight(string(fileBytes), style, formatter, lexer)
		if err != nil {
			log.Warn("Highlighting failed: ", err)
			return
		}

		if highlighted == nil {
			// No highlighting would be done, never mind
			return
		}

		reader.setText(*highlighted)
	}()
}

func highlightFromMemory(reader *Reader, style chroma.Style, formatter chroma.Formatter, lexer chroma.Lexer) {
	defer func() {
		reader.highlightingDone.Store(true)
		select {
		case reader.maybeDone <- true:
		default:
		}
	}()

	if lexer == nil {
		return
	}

	var byteCount int64
	reader.Lock()
	for _, line := range reader.lines {
		byteCount += int64(len(line.raw))
	}
	reader.Unlock()

	if byteCount > MAX_HIGHLIGHT_SIZE {
		log.Debug("File too large for highlighting: ", byteCount)
		return
	}

	textBuilder := strings.Builder{}
	reader.Lock()
	for _, line := range reader.lines {
		textBuilder.WriteString(line.raw)
		textBuilder.WriteString("\n")
	}
	reader.Unlock()

	highlighted, err := highlight(textBuilder.String(), style, formatter, lexer)
	if err != nil {
		log.Warn("Highlighting failed: ", err)
		return
	}

	if highlighted == nil {
		// No highlighting would be done, never mind
		return
	}

	reader.setText(*highlighted)
}

// createStatusUnlocked() assumes that its caller is holding the lock
func (reader *Reader) createStatusUnlocked(lastLine linenumbers.LineNumber) string {
	prefix := ""
	if reader.name != nil {
		prefix = path.Base(*reader.name) + ": "
	}

	if len(reader.lines) == 0 {
		return prefix + "<empty>"
	}

	if len(reader.lines) == 1 {
		return prefix + "1 line  100%"
	}

	percent := int(100 * float64(lastLine.AsOneBased()) / float64(len(reader.lines)))

	return fmt.Sprintf("%s%s lines  %d%%",
		prefix,
		linenumbers.LineNumberFromLength(len(reader.lines)).Format(),
		percent)
}

// GetLineCount returns the number of lines available for viewing
func (reader *Reader) GetLineCount() int {
	reader.Lock()
	defer reader.Unlock()

	return len(reader.lines)
}

// GetLine gets a line. If the requested line number is out of bounds, nil is returned.
func (reader *Reader) GetLine(lineNumber linenumbers.LineNumber) *Line {
	reader.Lock()
	defer reader.Unlock()

	if lineNumber.AsOneBased() > len(reader.lines) {
		return nil
	}
	return reader.lines[lineNumber.AsZeroBased()]
}

// GetLines gets the indicated lines from the input
//
// Overflow state will be didFit if we returned all lines we currently have, or
// didOverflow otherwise.
//
//revive:disable-next-line:unexported-return
func (reader *Reader) GetLines(firstLine linenumbers.LineNumber, wantedLineCount int) (*InputLines, overflowState) {
	reader.Lock()
	defer reader.Unlock()
	return reader.getLinesUnlocked(firstLine, wantedLineCount)
}

func (reader *Reader) getLinesUnlocked(firstLine linenumbers.LineNumber, wantedLineCount int) (*InputLines, overflowState) {
	if len(reader.lines) == 0 || wantedLineCount == 0 {
		return &InputLines{
				lines:      nil,
				firstLine:  firstLine,
				statusText: reader.createStatusUnlocked(firstLine),
			},
			didFit // Empty files always fit
	}

	lastLine := firstLine.NonWrappingAdd(wantedLineCount - 1)

	// Prevent reading past the end of the available lines
	maxLineNumber := *linenumbers.LineNumberFromLength(len(reader.lines))
	if lastLine.IsAfter(maxLineNumber) {
		lastLine = maxLineNumber

		// If one line was requested, then first and last should be exactly the
		// same, and we would get there by adding zero.
		firstLine = lastLine.NonWrappingAdd(1 - wantedLineCount)

		return reader.getLinesUnlocked(firstLine, firstLine.CountLinesTo(lastLine))
	}

	returnLines := reader.lines[firstLine.AsZeroBased() : lastLine.AsZeroBased()+1]
	overflow := didFit
	if len(returnLines) != len(reader.lines) {
		overflow = didOverflow // We're not returning all available lines
	}

	return &InputLines{
			lines:      returnLines,
			firstLine:  firstLine,
			statusText: reader.createStatusUnlocked(lastLine),
		},
		overflow
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

	reader.Lock()
	reader.lines = lines
	reader.replaced = true
	reader.Unlock()

	reader.done.Store(true)
	select {
	case reader.maybeDone <- true:
	default:
	}
	log.Trace("Reader done, contents explicitly set")

	select {
	case reader.moreLinesAdded <- true:
	default:
	}
}
