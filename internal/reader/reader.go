package reader

import (
	"bufio"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/walles/moor/v2/internal/linemetadata"
	"github.com/walles/moor/v2/internal/util"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	log "github.com/sirupsen/logrus"
)

// Files larger than this won't be highlighted
//
//revive:disable-next-line:var-naming
const MAX_HIGHLIGHT_SIZE int64 = 1024 * 1024

const DEFAULT_PAUSE_AFTER_LINES = 20_000

type ReaderOptions struct {
	// Format JSON input
	ShouldFormat bool

	// Pause after reading this many lines, unless told otherwise.
	// Tune at runtime using SetPauseAfterLines().
	//
	// nil means 20k lines.
	PauseAfterLines *int

	// If this is nil, you must call reader.SetStyleForHighlighting() later if
	// you want highlighting.
	Style *chroma.Style

	// If this is set, it will be used as the lexer for highlighting
	Lexer chroma.Lexer
}

type Reader interface {
	GetLineCount() int
	GetLine(index linemetadata.Index) *NumberedLine

	// This method will try to honor wantedLineCount over firstLine. This means
	// that the returned first line may be different from the requested one.
	GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines

	// False when paused. Showing the paused line count is confusing, because
	// the user might think that the number is the total line count, even though
	// we are not done yet.
	//
	// When we're not paused, the number will be constantly changing, indicating
	// that the counting is not done yet.
	ShouldShowLineCount() bool
}

// ReaderImpl reads a file into an array of strings.
//
// It does the reading in the background, and it returns parts of the read data
// upon request.
//
// This package provides query methods for the struct, no peeking!!
type ReaderImpl struct {
	sync.Mutex

	lines []*Line

	// Display name for the buffer. If not set, no buffer name will be shown.
	//
	// For files, this will be the file name. For our help text, this will be
	// "Help". For streams this will generally not be set.
	Name *string

	// If this is set, it will point out the file we are reading from. If this
	// is not set, we are not reading from a file.
	FileName *string

	// How many bytes have we read so far?
	bytesCount int64

	endsWithNewline bool

	Err error

	Done             *atomic.Bool
	HighlightingDone *atomic.Bool

	highlightingStyle chan chroma.Style

	// This channel expects to be read exactly once. All other uses will lead to
	// undefined behavior.
	doneWaitingForFirstByte chan bool

	// For telling the UI it should recheck the --quit-if-one-screen conditions.
	// Signalled when either highlighting is done or reading is done.
	MaybeDone chan bool

	MoreLinesAdded chan bool

	// Because we don't want to consume infinitely.
	//
	// Ref: https://github.com/walles/moor/issues/296
	pauseAfterLines        int
	pauseAfterLinesUpdated chan bool

	// PauseStatus is true if the reader is paused, false if it is not
	PauseStatus *atomic.Bool
}

// InputLines contains a number of lines from the reader, plus metadata
type InputLines struct {
	Lines []*NumberedLine

	// "monkey.txt: 1-23/45 51%"
	StatusText string
}

// Count lines in the original file and preallocate space for them.  Good
// performance improvement:
//
// go test -benchmem -benchtime=10s -run='^$' -bench 'ReadLargeFile'
func (reader *ReaderImpl) preAllocLines() {
	if reader.FileName == nil {
		return
	}

	if reader.GetLineCount() > 0 {
		// We already have lines, could be because we're tailing some file. Too
		// late for pre-allocation.
		return
	}

	lineCount, err := countLines(*reader.FileName)
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

// This is the reader's main function. It will be run in a goroutine. First it
// reads the stream until the end, then starts tailing.
func (reader *ReaderImpl) readStream(stream io.Reader, formatter chroma.Formatter, options ReaderOptions) {
	reader.consumeLinesFromStream(stream)

	t0 := time.Now()
	style := <-reader.highlightingStyle
	options.Style = &style
	highlightFromMemory(reader, formatter, options)
	log.Debug("highlightFromMemory() took ", time.Since(t0))

	reader.Done.Store(true)
	select {
	case reader.MaybeDone <- true:
	default:
	}

	// Tail the file if the stream is coming from a file.
	// Ref: https://github.com/walles/moor/issues/224
	err := reader.tailFile()
	if err != nil {
		log.Warn("Failed to tail file: ", err)
	}
}

// Pause if we should pause, otherwise not. Pausing means waiting for
// pauseAfterLinesUpdated to be signalled in SetPauseAfterLines().
func (reader *ReaderImpl) maybePause() {
	for {
		reader.Lock()
		shouldPause := len(reader.lines) >= reader.pauseAfterLines
		reader.Unlock()

		if !shouldPause {
			// Not there yet, no pause
			reader.setPauseStatus(false)
			return
		}

		reader.setPauseStatus(true)
		<-reader.pauseAfterLinesUpdated
	}
}

// This function will update the Reader struct. It is expected to run in a
// goroutine.
//
// It is used both during the initial read of the stream until it ends, and
// while tailing files for changes.
func (reader *ReaderImpl) consumeLinesFromStream(stream io.Reader) {
	reader.preAllocLines()

	inspectionReader := inspectionReader{base: stream}
	bufioReader := bufio.NewReader(&inspectionReader)
	completeLine := make([]byte, 0)

	t0 := time.Now()
	for {
		reader.maybePause()

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
			if reader.Err == nil {
				// Store the error unless it overwrites one we already have
				reader.Err = fmt.Errorf("error reading line from input stream: %w", err)
			}
			reader.Unlock()
		}

		if eof {
			break
		}

		if reader.Err != nil {
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

		// Reset our line buffer
		completeLine = completeLine[:0]

		// This is how to do a non-blocking write to a channel:
		// https://gobyexample.com/non-blocking-channel-operations
		select {
		case reader.MoreLinesAdded <- true:
		default:
			// Default case required for the write to be non-blocking
		}
	}

	if reader.FileName != nil {
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

	log.Info("Stream read in ", time.Since(t0), ", have ", reader.GetLineCount(), " lines")
}

func (reader *ReaderImpl) tailFile() error {
	reader.Lock()
	fileName := reader.FileName
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

// NewFromStream creates a new stream reader
//
// The name can be an empty string ("").
//
// If non-empty, the name will be displayed by the pager in the bottom left
// corner to help the user keep track of what is being paged.
//
// Note that you must call reader.SetStyleForHighlighting() after this to get
// highlighting.
func NewFromStream(name string, reader io.Reader, formatter chroma.Formatter, options ReaderOptions) (*ReaderImpl, error) {
	zReader, err := ZReader(reader)
	if err != nil {
		return nil, err
	}
	mReader := newReaderFromStream(zReader, nil, formatter, options)

	if len(name) > 0 {
		mReader.Lock()
		mReader.Name = &name
		mReader.Unlock()
	}

	if options.Lexer == nil {
		mReader.HighlightingDone.Store(true)
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
func newReaderFromStream(reader io.Reader, originalFileName *string, formatter chroma.Formatter, options ReaderOptions) *ReaderImpl {
	done := atomic.Bool{}
	done.Store(false)
	highlightingDone := atomic.Bool{}
	highlightingDone.Store(false)
	pauseStatus := atomic.Bool{}
	pauseStatus.Store(false)
	pauseAfterLines := DEFAULT_PAUSE_AFTER_LINES
	if options.PauseAfterLines != nil {
		pauseAfterLines = *options.PauseAfterLines
	}
	returnMe := ReaderImpl{
		// This needs to be size 1. If it would be 0, and we add more
		// lines while the pager is processing, the pager would miss
		// the lines added while it was processing.
		FileName: originalFileName,
		Name:     originalFileName,

		pauseAfterLines:        pauseAfterLines,
		pauseAfterLinesUpdated: make(chan bool, 1),

		PauseStatus: &pauseStatus,

		MoreLinesAdded:          make(chan bool, 1),
		MaybeDone:               make(chan bool, 1),
		highlightingStyle:       make(chan chroma.Style, 1),
		doneWaitingForFirstByte: make(chan bool, 1),
		HighlightingDone:        &highlightingDone,
		Done:                    &done,
	}

	go func() {
		defer func() {
			PanicHandler("newReaderFromStream()/readStream()", recover(), debug.Stack())
		}()

		returnMe.readStream(reader, formatter, options)
	}()

	return &returnMe
}

// Testing only!! May or may not hang if run in real world scenarios.
//
// NewFromTextForTesting creates a Reader from a block of text.
//
// First parameter is the name of this Reader. This name will be displayed by
// Moor in the bottom left corner of the screen.
//
// Calling Wait() on this Reader will always return immediately, no
// asynchronous ops will be performed.
func NewFromTextForTesting(name string, text string) *ReaderImpl {
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
	returnMe := &ReaderImpl{
		lines:                   lines,
		Done:                    &done,
		HighlightingDone:        &highlightingDone,
		doneWaitingForFirstByte: make(chan bool, 1),
	}
	if name != "" {
		returnMe.Name = &name
	}

	return returnMe
}

// Duplicate of moor/moor.go:TryOpen
func TryOpen(filename string) error {
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

// NewFromFilename creates a new file reader.
//
// If options.Lexer is nil it will be determined from the input file name.
//
// If options.Style is nil, you must call reader.SetStyleForHighlighting() later
// to get highlighting.
//
// The Reader will try to uncompress various compressed file format, and also
// apply highlighting to the file using Chroma:
// https://github.com/alecthomas/chroma
func NewFromFilename(filename string, formatter chroma.Formatter, options ReaderOptions) (*ReaderImpl, error) {
	fileError := TryOpen(filename)
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
		returnMe.HighlightingDone.Store(true)
	}

	if options.Style != nil {
		returnMe.SetStyleForHighlighting(*options.Style)
	}

	return returnMe, nil
}

// Wait for reader to finish reading and highlighting. Used by tests.
func (reader *ReaderImpl) Wait() error {
	// Wait for our goroutine to finish
	//revive:disable-next-line:empty-block
	for !reader.Done.Load() {
		if reader.PauseStatus.Load() {
			// We want more lines
			reader.SetPauseAfterLines(reader.GetLineCount() * 2)
		}
	}
	//revive:disable-next-line:empty-block
	for !reader.HighlightingDone.Load() {
	}

	reader.Lock()
	defer reader.Unlock()
	return reader.Err
}

func textAsString(reader *ReaderImpl, shouldFormat bool) string {
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
func highlightFromMemory(reader *ReaderImpl, formatter chroma.Formatter, options ReaderOptions) {
	defer func() {
		reader.HighlightingDone.Store(true)
		select {
		case reader.MaybeDone <- true:
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

	highlighted, err := Highlight(text, *options.Style, formatter, options.Lexer)
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
func (reader *ReaderImpl) createStatusUnlocked(lastLine linemetadata.Index) string {
	filename := ""
	if reader.Name != nil {
		filename = filepath.Base(*reader.Name)
	}

	if len(reader.lines) == 0 {
		empty := "<empty>"
		if len(filename) > 0 {
			return filename + ": " + empty
		}
		return empty
	}

	linesCount := ""
	percent := ""
	if len(reader.lines) == 1 {
		linesCount = "1 line"
		percent = "100%"
	} else {
		// More than one line
		linesCount = util.FormatInt(len(reader.lines)) + " lines"
		percent = fmt.Sprintf("%.0f%%", math.Floor(100*float64(lastLine.Index()+1)/float64(len(reader.lines))))
	}

	if !reader.ShouldShowLineCount() {
		linesCount = ""
	}

	return_me := ""
	if len(filename) > 0 {
		return_me = filename
	}

	if len(linesCount) > 0 {
		if len(filename) > 0 {
			return_me += ": "
		}
		return_me += linesCount
	}

	if len(percent) > 0 {
		if len(return_me) > 0 {
			return_me += "  "
		}
		return_me += percent
	}

	return return_me
}

// Wait for the first line to be read.
//
// Used for making sudo work:
// https://github.com/walles/moor/issues/199
func (reader *ReaderImpl) AwaitFirstByte() {
	<-reader.doneWaitingForFirstByte
}

// GetLineCount returns the number of lines available for viewing
func (reader *ReaderImpl) GetLineCount() int {
	reader.Lock()
	defer reader.Unlock()

	return len(reader.lines)
}

func (reader *ReaderImpl) ShouldShowLineCount() bool {
	if reader.Done.Load() {
		// We are done, the number won't change, show it!
		return true
	}

	if !reader.PauseStatus.Load() {
		// Reading in progress, number is constantly changing so it's
		// obvious we aren't done yet. Show it!
		return true
	}

	return false
}

// GetLine gets a line. If the requested line number is out of bounds, nil is returned.
func (reader *ReaderImpl) GetLine(index linemetadata.Index) *NumberedLine {
	reader.Lock()
	defer reader.Unlock()

	if index.Index() >= reader.pauseAfterLines-DEFAULT_PAUSE_AFTER_LINES/2 {
		// Getting close(ish) to the pause threshold, bump it up. The Max()
		// construct is to handle the case when the add overflows.
		reader.pauseAfterLines = slices.Max([]int{
			reader.pauseAfterLines + DEFAULT_PAUSE_AFTER_LINES/2,
			reader.pauseAfterLines})
		select {
		case reader.pauseAfterLinesUpdated <- true:
		default:
			// Default case required for the write to be non-blocking
		}
	}

	if !index.IsWithinLength(len(reader.lines)) {
		return nil
	}
	return &NumberedLine{
		Index:  index,
		Number: linemetadata.NumberFromZeroBased(index.Index()),
		Line:   reader.lines[index.Index()],
	}
}

// GetLines gets the indicated lines from the input
//
//revive:disable-next-line:unexported-return
func (reader *ReaderImpl) GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	reader.Lock()
	defer reader.Unlock()
	return reader.getLinesUnlocked(firstLine, wantedLineCount)
}

func (reader *ReaderImpl) getLinesUnlocked(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	if len(reader.lines) == 0 || wantedLineCount == 0 {
		return &InputLines{
			StatusText: reader.createStatusUnlocked(firstLine),
		}
	}

	lastLine := firstLine.NonWrappingAdd(wantedLineCount - 1)

	// Prevent reading past the end of the available lines
	maxLineIndex := *linemetadata.IndexFromLength(len(reader.lines))
	if lastLine.IsAfter(maxLineIndex) {
		lastLine = maxLineIndex

		// If one line was requested, then first and last should be exactly the
		// same, and we would get there by adding zero.
		firstLine = lastLine.NonWrappingAdd(1 - wantedLineCount)

		return reader.getLinesUnlocked(firstLine, firstLine.CountLinesTo(lastLine))
	}

	notNumberedReturnLines := reader.lines[firstLine.Index() : lastLine.Index()+1]
	returnLines := make([]*NumberedLine, 0, len(notNumberedReturnLines))
	for loopIndex, line := range notNumberedReturnLines {
		lineIndex := firstLine.NonWrappingAdd(loopIndex)
		returnLines = append(returnLines, &NumberedLine{
			Index:  lineIndex,
			Number: linemetadata.NumberFromZeroBased(lineIndex.Index()),
			Line:   line,
		})
	}

	return &InputLines{
		Lines:      returnLines,
		StatusText: reader.createStatusUnlocked(lastLine),
	}
}

func (reader *ReaderImpl) PumpToStdout() {
	const wantedLineCount = 100
	firstNotPrintedLine := linemetadata.Index{}

	drainLines := func() bool {
		lines := reader.GetLines(firstNotPrintedLine, wantedLineCount)
		var firstReturnedIndex linemetadata.Index
		if len(lines.Lines) > 0 {
			firstReturnedIndex = lines.Lines[0].Index
		}

		// Print the lines we got
		printed := false
		for loopIndex, line := range lines.Lines {
			lineIndex := firstReturnedIndex.NonWrappingAdd(loopIndex)
			if lineIndex.IsBefore(firstNotPrintedLine) {
				continue
			}

			fmt.Println(line.Line.raw)
			printed = true
			firstNotPrintedLine = lineIndex.NonWrappingAdd(1)
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
		case <-reader.MoreLinesAdded:
			continue
		case <-reader.MaybeDone:
			done = true
		}
	}

	// Print any remaining lines
	drainAllLines()
}

// Replace reader contents with the given text and mark as done
func (reader *ReaderImpl) setText(text string) {
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

	reader.Done.Store(true)
	select {
	case reader.MaybeDone <- true:
	default:
	}
	log.Trace("Reader done, contents explicitly set")

	select {
	case reader.MoreLinesAdded <- true:
	default:
	}
}

func (reader *ReaderImpl) setPauseStatus(paused bool) {
	if !reader.PauseStatus.CompareAndSwap(!paused, paused) {
		// Pause status already had that value, we're done
		return
	}

	log.Debugf("Reader pause status changed to %t", paused)
}

func (reader *ReaderImpl) SetPauseAfterLines(lines int) {
	if lines < 0 {
		log.Warnf("Tried to set pause-after-lines to %d, ignoring", lines)
		return
	}

	log.Trace("Setting pause-after-lines to ", lines, "...")

	reader.Lock()
	reader.pauseAfterLines = lines
	reader.Unlock()

	// Notify the reader that the pause-after-lines value has been updated. Will
	// be noticed in the maybePause() function.
	select {
	case reader.pauseAfterLinesUpdated <- true:
	default:
		// Default case required for the write to be non-blocking
	}
}

func (reader *ReaderImpl) SetStyleForHighlighting(style chroma.Style) {
	reader.highlightingStyle <- style
}
