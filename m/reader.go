package m

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/walles/moar/m/linemetadata"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	log "github.com/sirupsen/logrus"
)

// Files larger than this won't be highlighted
//
//revive:disable-next-line:var-naming
const MAX_HIGHLIGHT_SIZE int64 = 1024 * 1024

type ReaderOptions struct {
	// Format JSON input
	ShouldFormat bool

	// If this is nil, you must call reader.SetStyleForHighlighting() later if
	// you want highlighting.
	Style *chroma.Style

	// If this is set, it will be used as the lexer for highlighting
	Lexer chroma.Lexer
}

// Reader reads a file into an array of strings.
//
// It does the reading in the background, and it returns parts of the read data
// upon request.
//
// This package provides query methods for the struct, no peeking!!
type Reader struct {
	sync.Mutex

	lines []*Line

	// Display name for the buffer. If not set, no buffer name will be shown.
	//
	// For files, this will be the file name. For our help text, this will be
	// "Help". For streams this will generally not be set.
	name *string

	// If this is set, it will point out the file we are reading from. If this
	// is not set, we are not reading from a file.
	fileName *string

	// How many bytes have we read so far?
	bytesCount int64

	endsWithNewline bool

	err error

	done             *atomic.Bool
	highlightingDone *atomic.Bool

	highlightingStyle chan chroma.Style

	// This channel expects to be read exactly once. All other uses will lead to
	// undefined behavior.
	doneWaitingForFirstByte chan bool

	// For telling the UI it should recheck the --quit-if-one-screen conditions.
	// Signalled when either highlighting is done or reading is done.
	maybeDone chan bool

	moreLinesAdded chan bool
}

// InputLines contains a number of lines from the reader, plus metadata
type InputLines struct {
	lines []*NumberedLine

	// Line number of the first line returned
	firstLine linemetadata.Index

	// "monkey.txt: 1-23/45 51%"
	statusText string
}

// Count lines in the original file and preallocate space for them.  Good
// performance improvement:
//
// go test -benchmem -benchtime=10s -run='^$' -bench 'ReadLargeFile'
func (reader *Reader) preAllocLines() {
	if reader.fileName == nil {
		return
	}

	if reader.GetLineCount() > 0 {
		// We already have lines, could be because we're tailing some file. Too
		// late for pre-allocation.
		return
	}

	lineCount, err := countLines(*reader.fileName)
	if err != nil {
		log.Warn("Line counting failed: ", err)
		return
	}

	reader.Lock()
	defer reader.Unlock()

	if len(reader.lines) != 0 {
		// I don't understand how this could happen.
		log.Warnf("Already had %d lines by the time counting was done", len(reader.lines))
		return
	}

	// We had no lines since before, this is the expected happy path.
	reader.lines = make([]*Line, 0, lineCount)
}

func (reader *Reader) readStream(stream io.Reader, formatter chroma.Formatter, options ReaderOptions) {
	reader.consumeLinesFromStream(stream)

	t0 := time.Now()
	style := <-reader.highlightingStyle
	options.Style = &style
	highlightFromMemory(reader, formatter, options)
	log.Debug("highlightFromMemory() took ", time.Since(t0))

	reader.done.Store(true)
	select {
	case reader.maybeDone <- true:
	default:
	}

	// Tail the file if the stream is coming from a file.
	// Ref: https://github.com/walles/moar/issues/224
	err := reader.tailFile()
	if err != nil {
		log.Warn("Failed to tail file: ", err)
	}
}

// This function will update the Reader struct. It is expected to run in a
// goroutine.
func (reader *Reader) consumeLinesFromStream(stream io.Reader) {
	reader.preAllocLines()

	inspectionReader := inspectionReader{base: stream}
	bufioReader := bufio.NewReader(&inspectionReader)
	completeLine := make([]byte, 0)

	t0 := time.Now()
	for {
		keepReadingLine := true
		eof := false

		var lineBytes []byte
		var err error
		for keepReadingLine {
			lineBytes, keepReadingLine, err = bufioReader.ReadLine()

			if err == nil {
				select {
				// Async write, we probably already wrote to it during the last
				// iteration
				case reader.doneWaitingForFirstByte <- true:
				default:
				}

				completeLine = append(completeLine, lineBytes...)
				continue
			}

			// Something went wrong

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
		if len(reader.lines) > 0 && !reader.endsWithNewline {
			// The last line didn't end with a newline, append to it
			newLineString = reader.lines[len(reader.lines)-1].raw + newLineString
			newLine = NewLine(newLineString)
			reader.lines[len(reader.lines)-1] = &newLine
		} else {
			reader.lines = append(reader.lines, &newLine)
		}
		reader.endsWithNewline = true
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

	if reader.fileName != nil {
		reader.Lock()
		reader.bytesCount += inspectionReader.bytesCount
		reader.Unlock()
	}

	// If the stream was empty we never got any first byte. Make sure people
	// stop waiting in this case. Async write since it might already have been
	// written to.
	select {
	case reader.doneWaitingForFirstByte <- true:
	default:
	}

	reader.endsWithNewline = inspectionReader.endedWithNewline

	log.Info("Stream read in ", time.Since(t0))
}

func (reader *Reader) tailFile() error {
	reader.Lock()
	fileName := reader.fileName
	reader.Unlock()
	if fileName == nil {
		return nil
	}

	log.Debugf("Tailing file %s", *fileName)

	for {
		// NOTE: We could use something like
		// https://github.com/fsnotify/fsnotify instead of sleeping and polling
		// here.
		time.Sleep(1 * time.Second)

		fileStats, err := os.Stat(*fileName)
		if err != nil {
			log.Debugf("Failed to stat file %s while tailing, giving up: %s", *fileName, err.Error())
			return nil
		}

		reader.Lock()
		bytesCount := reader.bytesCount
		reader.Unlock()

		if bytesCount == -1 {
			log.Debugf("Bytes count unknown for %s, stop tailing", *fileName)
			return nil
		}

		if fileStats.Size() == bytesCount {
			log.Tracef("File %s unchanged at %d bytes, continue tailing", *fileName, fileStats.Size())
			continue
		}

		if fileStats.Size() < bytesCount {
			log.Debugf("File %s shrunk from %d to %d bytes, stop tailing",
				*fileName, bytesCount, fileStats.Size())
			return nil
		}

		// File grew, read the new lines
		stream, _, err := ZOpen(*fileName)
		if err != nil {
			log.Debugf("Failed to open file %s for re-reading while tailing: %s", *fileName, err.Error())
			return nil
		}

		seekable, ok := stream.(io.ReadSeekCloser)
		if !ok {
			err = stream.Close()
			if err != nil {
				log.Debugf("Giving up on tailing, failed to close non-seekable stream from %s: %s", *fileName, err.Error())
				return nil
			}
			log.Debugf("Giving up on tailing, file %s is not seekable", *fileName)
			return nil
		}
		_, err = seekable.Seek(bytesCount, io.SeekStart)
		if err != nil {
			log.Debugf("Failed to seek in file %s while tailing: %s", *fileName, err.Error())
			return nil
		}

		log.Tracef("File %s up from %d bytes to %d bytes, reading more lines...", *fileName, bytesCount, fileStats.Size())

		reader.consumeLinesFromStream(seekable)
		err = seekable.Close()
		if err != nil {
			// This can lead to file handle leaks
			return fmt.Errorf("failed to close file %s after tailing: %w", *fileName, err)
		}
	}
}

// NewReaderFromStream creates a new stream reader
//
// The name can be an empty string ("").
//
// If non-empty, the name will be displayed by the pager in the bottom left
// corner to help the user keep track of what is being paged.
//
// Note that you must call reader.SetStyleForHighlighting() after this to get
// highlighting.
func NewReaderFromStream(name string, reader io.Reader, formatter chroma.Formatter, options ReaderOptions) (*Reader, error) {
	zReader, err := ZReader(reader)
	if err != nil {
		return nil, err
	}
	mReader := newReaderFromStream(zReader, nil, formatter, options)

	if len(name) > 0 {
		mReader.Lock()
		mReader.name = &name
		mReader.Unlock()
	}

	if options.Lexer == nil {
		mReader.highlightingDone.Store(true)
	}

	if options.Style != nil {
		mReader.SetStyleForHighlighting(*options.Style)
	}

	return mReader, nil
}

// newReaderFromStream creates a new stream reader
//
// originalFileName is used for counting the lines in the file. nil for
// don't-know (streams) or not countable (compressed files). The line count is
// then used for pre-allocating the lines slice, which improves large file
// loading performance.
//
// If lexer is set, the file will be highlighted after being fully read.
//
// Whatever data we get from the reader, that's what we'll have. Or in other
// words, if the input needs to be decompressed, do that before coming here.
//
// Note that you must call reader.SetStyleForHighlighting() after this to get
// highlighting.
func newReaderFromStream(reader io.Reader, originalFileName *string, formatter chroma.Formatter, options ReaderOptions) *Reader {
	done := atomic.Bool{}
	done.Store(false)
	highlightingDone := atomic.Bool{}
	highlightingDone.Store(false)
	returnMe := Reader{
		// This needs to be size 1. If it would be 0, and we add more
		// lines while the pager is processing, the pager would miss
		// the lines added while it was processing.
		fileName:                originalFileName,
		name:                    originalFileName,
		moreLinesAdded:          make(chan bool, 1),
		maybeDone:               make(chan bool, 1),
		highlightingStyle:       make(chan chroma.Style, 1),
		doneWaitingForFirstByte: make(chan bool, 1),
		highlightingDone:        &highlightingDone,
		done:                    &done,
	}

	go func() {
		defer func() {
			panicHandler("newReaderFromStream()/readStream()", recover(), debug.Stack())
		}()

		returnMe.readStream(reader, formatter, options)
	}()

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
		lines:                   lines,
		done:                    &done,
		highlightingDone:        &highlightingDone,
		doneWaitingForFirstByte: make(chan bool, 1),
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

	reader, _, err := ZOpen(filename)
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
	t0 := time.Now()
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

	t1 := time.Now()
	if count == 0 {
		log.Debug("Counted ", count, " lines in ", t1.Sub(t0))
	} else {
		log.Debug("Counted ", count, " lines in ", t1.Sub(t0), " at ", t1.Sub(t0)/time.Duration(count), "/line")
	}
	return count, nil
}

// NewReaderFromFilename creates a new file reader.
//
// If options.Lexer is nil it will be determined from the input file name.
//
// If options.Style is nil, you must call reader.SetStyleForHighlighting() later
// to get highlighting.
//
// The Reader will try to uncompress various compressed file format, and also
// apply highlighting to the file using Chroma:
// https://github.com/alecthomas/chroma
func NewReaderFromFilename(filename string, formatter chroma.Formatter, options ReaderOptions) (*Reader, error) {
	fileError := tryOpen(filename)
	if fileError != nil {
		return nil, fileError
	}

	stream, highlightingFilename, err := ZOpen(filename)
	if err != nil {
		return nil, err
	}

	if options.Lexer == nil {
		options.Lexer = lexers.Match(highlightingFilename)
	}

	returnMe := newReaderFromStream(stream, &highlightingFilename, formatter, options)

	if options.Lexer == nil {
		returnMe.highlightingDone.Store(true)
	}

	if options.Style != nil {
		returnMe.SetStyleForHighlighting(*options.Style)
	}

	return returnMe, nil
}

func textAsString(reader *Reader, shouldFormat bool) string {
	reader.Lock()

	text := strings.Builder{}
	for _, line := range reader.lines {
		text.WriteString(line.raw)
		text.WriteString("\n")
	}
	result := text.String()
	reader.Unlock()

	var jsonData any
	err := json.Unmarshal([]byte(result), &jsonData)
	if err != nil {
		// Not JSON, return the text as-is
		return result
	}

	if !shouldFormat {
		log.Info("Try the --reformat flag for automatic JSON reformatting")
		return result
	}

	// Pretty print the JSON
	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		log.Debug("Failed to pretty print JSON: ", err)
		return result
	}

	log.Debug("Got the --reformat flag, reformatted JSON input")
	return string(prettyJSON)
}

func isXml(text string) bool {
	err := xml.Unmarshal([]byte(text), new(any))
	return err == nil
}

// We expect this to be executed in a goroutine
func highlightFromMemory(reader *Reader, formatter chroma.Formatter, options ReaderOptions) {
	defer func() {
		reader.highlightingDone.Store(true)
		select {
		case reader.maybeDone <- true:
		default:
		}
	}()

	// Is the buffer small enough?
	var byteCount int64
	reader.Lock()
	for _, line := range reader.lines {
		byteCount += int64(len(line.raw))

		if byteCount > MAX_HIGHLIGHT_SIZE {
			log.Info("File too large for highlighting: ", byteCount)
			reader.Unlock()
			return
		}
	}
	reader.Unlock()

	text := textAsString(reader, options.ShouldFormat)

	if len(text) == 0 {
		log.Debug("Buffer is empty, not highlighting")
		return
	}

	if options.Lexer == nil && json.Valid([]byte(text)) {
		log.Info("Buffer is valid JSON, highlighting as JSON")
		options.Lexer = lexers.Get("json")
	} else if options.Lexer == nil && isXml(text) {
		log.Info("Buffer is valid XML, highlighting as XML")
		options.Lexer = lexers.Get("xml")
	}

	if options.Lexer == nil {
		log.Debug("No lexer set, not highlighting")
		return
	}

	if options.Style == nil {
		log.Debug("No style set, not highlighting")
		return
	}

	if formatter == nil {
		log.Debug("No formatter set, not highlighting")
		return
	}

	highlighted, err := highlight(text, *options.Style, formatter, options.Lexer)
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
func (reader *Reader) createStatusUnlocked(lastLine linemetadata.Index) string {
	prefix := ""
	if reader.name != nil {
		prefix = filepath.Base(*reader.name) + ": "
	}

	if len(reader.lines) == 0 {
		return prefix + "<empty>"
	}

	if len(reader.lines) == 1 {
		return prefix + "1 line  100%"
	}

	percent := int(100 * float64(lastLine.Index()+1) / float64(len(reader.lines)))

	return fmt.Sprintf("%s%s lines  %d%%",
		prefix,
		linemetadata.IndexFromLength(len(reader.lines)).Format(),
		percent)
}

// Wait for the first line to be read.
//
// Used for making sudo work:
// https://github.com/walles/moar/issues/199
func (reader *Reader) AwaitFirstByte() {
	<-reader.doneWaitingForFirstByte
}

// GetLineCount returns the number of lines available for viewing
func (reader *Reader) GetLineCount() int {
	reader.Lock()
	defer reader.Unlock()

	return len(reader.lines)
}

// GetLine gets a line. If the requested line number is out of bounds, nil is returned.
func (reader *Reader) GetLine(index linemetadata.Index) *NumberedLine {
	reader.Lock()
	defer reader.Unlock()

	if index.IsWithinLength(len(reader.lines)) {
		return nil
	}
	return &NumberedLine{
		number: linemetadata.NumberFromZeroBased(index.Index()),
		line:   reader.lines[index.Index()],
	}
}

// GetLines gets the indicated lines from the input
//
//revive:disable-next-line:unexported-return
func (reader *Reader) GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	reader.Lock()
	defer reader.Unlock()
	return reader.getLinesUnlocked(firstLine, wantedLineCount)
}

func (reader *Reader) getLinesUnlocked(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	if len(reader.lines) == 0 || wantedLineCount == 0 {
		return &InputLines{
			lines:      nil,
			firstLine:  firstLine,
			statusText: reader.createStatusUnlocked(firstLine),
		}
	}

	lastLine := firstLine.NonWrappingAdd(wantedLineCount - 1)

	// Prevent reading past the end of the available lines
	maxLineNumber := *linemetadata.IndexFromLength(len(reader.lines))
	if lastLine.IsAfter(maxLineNumber) {
		lastLine = maxLineNumber

		// If one line was requested, then first and last should be exactly the
		// same, and we would get there by adding zero.
		firstLine = lastLine.NonWrappingAdd(1 - wantedLineCount)

		return reader.getLinesUnlocked(firstLine, firstLine.CountLinesTo(lastLine))
	}

	notNumberedReturnLines := reader.lines[firstLine.Index() : lastLine.Index()+1]
	returnLines := make([]*NumberedLine, 0, len(notNumberedReturnLines))
	for index, line := range notNumberedReturnLines {
		lineNumber := linemetadata.NumberFromZeroBased(firstLine.NonWrappingAdd(index).Index())
		returnLines = append(returnLines, &NumberedLine{
			number: lineNumber,
			line:   line,
		})
	}

	return &InputLines{
		lines:      returnLines,
		firstLine:  firstLine,
		statusText: reader.createStatusUnlocked(lastLine),
	}
}

func (reader *Reader) PumpToStdout() {
	const wantedLineCount = 100
	firstNotPrintedLine := linemetadata.Index{}

	drainLines := func() bool {
		lines := reader.GetLines(firstNotPrintedLine, wantedLineCount)

		// Print the lines we got
		printed := false
		for index, line := range lines.lines {
			lineNumber := lines.firstLine.NonWrappingAdd(index)
			if lineNumber.IsBefore(firstNotPrintedLine) {
				continue
			}

			fmt.Println(line.line.raw)
			printed = true
			firstNotPrintedLine = lineNumber.NonWrappingAdd(1)
		}

		return printed
	}

	drainAllLines := func() {
		for drainLines() {
			// Loop here until nothing was printed
		}
	}

	done := false
	for !done {
		drainAllLines()

		select {
		case <-reader.moreLinesAdded:
			continue
		case <-reader.maybeDone:
			done = true
		}
	}

	// Print any remaining lines
	drainAllLines()
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

func (reader *Reader) SetStyleForHighlighting(style chroma.Style) {
	reader.highlightingStyle <- style
}
