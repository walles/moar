package m

import (
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/walles/moar/m/linenumbers"
	"gotest.tools/v3/assert"
)

//revive:disable:empty-block

const samplesDir = "../sample-files"

func testGetLineCount(t *testing.T, reader *Reader) {
	if strings.Contains(*reader.name, "compressed") {
		// We are no good at counting lines of compressed files, never mind
		return
	}

	cmd := exec.Command("wc", "-l", *reader.name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Error("Error calling wc -l to count lines of", *reader.name, err)
	}

	wcNumberString := strings.Split(strings.TrimSpace(string(output)), " ")[0]
	wcLineCount, err := strconv.Atoi(wcNumberString)
	if err != nil {
		t.Error("Error counting lines of", *reader.name, err)
	}

	if strings.HasSuffix(*reader.name, "/line-without-newline.txt") {
		// "wc -l" thinks this file contains zero lines
		wcLineCount = 1
	} else if strings.HasSuffix(*reader.name, "/two-lines-no-trailing-newline.txt") {
		// "wc -l" thinks this file contains one line
		wcLineCount = 2
	}

	if reader.GetLineCount() != wcLineCount {
		t.Errorf("Got %d lines from the reader but %d lines from wc -l: <%s>",
			reader.GetLineCount(), wcLineCount, *reader.name)
	}

	countLinesCount, err := countLines(*reader.name)
	assert.NilError(t, err)
	if countLinesCount != uint64(wcLineCount) {
		t.Errorf("Got %d lines from wc -l, but %d lines from our countLines() function", wcLineCount, countLinesCount)
	}
}

func testGetLines(t *testing.T, reader *Reader) {
	lines, _ := reader.GetLines(linenumbers.LineNumber{}, 10)
	if len(lines.lines) > 10 {
		t.Errorf("Asked for 10 lines, got too many: %d", len(lines.lines))
	}

	if len(lines.lines) < 10 {
		// No good plan for how to test short files, more than just
		// querying them, which we just did
		return
	}

	// Test clipping at the end
	lines, _ = reader.GetLines(linenumbers.LineNumberMax(), 10)
	if len(lines.lines) != 10 {
		t.Errorf("Asked for 10 lines but got %d", len(lines.lines))
		return
	}

	startOfLastSection := lines.firstLine
	lines, _ = reader.GetLines(startOfLastSection, 10)
	if lines.firstLine != startOfLastSection {
		t.Errorf("Expected start line %d when asking for the last 10 lines, got %s",
			startOfLastSection, lines.firstLine.Format())
		return
	}
	if len(lines.lines) != 10 {
		t.Errorf("Expected 10 lines when asking for the last 10 lines, got %d",
			len(lines.lines))
		return
	}

	lines, _ = reader.GetLines(startOfLastSection.NonWrappingAdd(1), 10)
	if lines.firstLine != startOfLastSection {
		t.Errorf("Expected start line %d when asking for the last+1 10 lines, got %s",
			startOfLastSection, lines.firstLine.Format())
		return
	}
	if len(lines.lines) != 10 {
		t.Errorf("Expected 10 lines when asking for the last+1 10 lines, got %d",
			len(lines.lines))
		return
	}

	lines, _ = reader.GetLines(startOfLastSection.NonWrappingAdd(-1), 10)
	if lines.firstLine != startOfLastSection.NonWrappingAdd(-1) {
		t.Errorf("Expected start line %d when asking for the last-1 10 lines, got %s",
			startOfLastSection, lines.firstLine.Format())
		return
	}
	if len(lines.lines) != 10 {
		t.Errorf("Expected 10 lines when asking for the last-1 10 lines, got %d",
			len(lines.lines))
		return
	}
}

func getTestFiles(t *testing.T) []string {
	files, err := os.ReadDir(samplesDir)
	assert.NilError(t, err)

	var filenames []string
	for _, file := range files {
		filenames = append(filenames, path.Join(samplesDir, file.Name()))
	}

	return filenames
}

// Wait for reader to finish reading and highlighting. Used by tests.
func (r *Reader) _wait() error {
	// Wait for our goroutine to finish
	//revive:disable-next-line:empty-block
	for !r.done.Load() {
	}
	//revive:disable-next-line:empty-block
	for !r.highlightingDone.Load() {
	}

	r.Lock()
	defer r.Unlock()
	return r.err
}

func TestGetLines(t *testing.T) {
	for _, file := range getTestFiles(t) {
		reader, err := NewReaderFromFilename(file, *styles.Get("native"), formatters.TTY16m, nil)
		if err != nil {
			t.Errorf("Error opening file <%s>: %s", file, err.Error())
			continue
		}
		if err := reader._wait(); err != nil {
			t.Errorf("Error reading file <%s>: %s", file, err.Error())
			continue
		}

		t.Run(file, func(t *testing.T) {
			testGetLines(t, reader)
			testGetLineCount(t, reader)
			testHighlightingLineCount(t, file)
		})
	}
}

func testHighlightingLineCount(t *testing.T, filenameWithPath string) {
	// This won't work on compressed files
	if strings.HasSuffix(filenameWithPath, ".xz") {
		return
	}
	if strings.HasSuffix(filenameWithPath, ".bz2") {
		return
	}
	if strings.HasSuffix(filenameWithPath, ".gz") {
		return
	}
	if strings.HasSuffix(filenameWithPath, ".zst") {
		return
	}
	if strings.HasSuffix(filenameWithPath, ".zstd") {
		return
	}

	// Load the unformatted file
	rawBytes, err := os.ReadFile(filenameWithPath)
	assert.NilError(t, err)
	rawContents := string(rawBytes)

	// Count its lines
	rawLinefeedsCount := strings.Count(rawContents, "\n")
	rawRunes := []rune(rawContents)
	rawFileEndsWithNewline := true // Special case empty files
	if len(rawRunes) > 0 {
		rawFileEndsWithNewline = rawRunes[len(rawRunes)-1] == '\n'
	}
	rawLinesCount := rawLinefeedsCount
	if !rawFileEndsWithNewline {
		rawLinesCount++
	}

	// Then load the same file using one of our Readers
	reader, err := NewReaderFromFilename(filenameWithPath, *styles.Get("native"), formatters.TTY16m, nil)
	assert.NilError(t, err)
	err = reader._wait()
	assert.NilError(t, err)

	highlightedLinesCount := reader.GetLineCount()
	assert.Equal(t, rawLinesCount, highlightedLinesCount)
}

func TestGetLongLine(t *testing.T) {
	file := "../sample-files/very-long-line.txt"
	reader, err := NewReaderFromFilename(file, *styles.Get("native"), formatters.TTY16m, nil)
	assert.NilError(t, err)
	assert.NilError(t, reader._wait())

	lines, overflow := reader.GetLines(linenumbers.LineNumber{}, 5)
	assert.Equal(t, lines.firstLine, linenumbers.LineNumber{})
	assert.Equal(t, len(lines.lines), 1)

	// This fits because we got all (one) input lines. Given the line length the
	// line is unlikely to fit on screen, but that's not what this didFit is
	// about.
	assert.Equal(t, overflow, didFit)

	line := lines.lines[0]
	assert.Assert(t, strings.HasPrefix(line.Plain(nil), "1 2 3 4"), "<%s>", line)
	assert.Assert(t, strings.HasSuffix(line.Plain(nil), "0123456789"), line)

	assert.Equal(t, len(line.Plain(nil)), 100021)
}

func getReaderWithLineCount(totalLines int) *Reader {
	return NewReaderFromText("", strings.Repeat("x\n", totalLines))
}

func testStatusText(t *testing.T, fromLine linenumbers.LineNumber, toLine linenumbers.LineNumber, totalLines int, expected string) {
	testMe := getReaderWithLineCount(totalLines)
	linesRequested := fromLine.CountLinesTo(toLine)
	lines, _ := testMe.GetLines(fromLine, linesRequested)
	statusText := lines.statusText
	assert.Equal(t, statusText, expected)
}

func TestStatusText(t *testing.T) {
	testStatusText(t, linenumbers.LineNumber{}, linenumbers.LineNumberFromOneBased(10), 20, "20 lines  50%")
	testStatusText(t, linenumbers.LineNumber{}, linenumbers.LineNumberFromOneBased(5), 5, "5 lines  100%")
	testStatusText(t,
		linenumbers.LineNumberFromOneBased(998),
		linenumbers.LineNumberFromOneBased(999),
		1000,
		"1000 lines  99%")

	testStatusText(t, linenumbers.LineNumber{}, linenumbers.LineNumber{}, 0, "<empty>")
	testStatusText(t, linenumbers.LineNumber{}, linenumbers.LineNumber{}, 1, "1 line  100%")

	// Test with filename
	testMe, err := NewReaderFromFilename(samplesDir+"/empty", *styles.Get("native"), formatters.TTY16m, nil)
	assert.NilError(t, err)
	assert.NilError(t, testMe._wait())

	line, overflow := testMe.GetLines(linenumbers.LineNumber{}, 0)
	if line.lines != nil {
		t.Error("line.lines is should have been nil when reading from an empty stream")
	}
	assert.Equal(t, line.statusText, "empty: <empty>")
	assert.Equal(t, overflow, didFit) // Empty always fits
}

func testCompressedFile(t *testing.T, filename string) {
	filenameWithPath := path.Join(samplesDir, filename)
	reader, e := NewReaderFromFilename(filenameWithPath, *styles.Get("native"), formatters.TTY16m, nil)
	if e != nil {
		t.Errorf("Error opening file <%s>: %s", filenameWithPath, e.Error())
		panic(e)
	}
	assert.NilError(t, reader._wait())

	lines, _ := reader.GetLines(linenumbers.LineNumber{}, 5)
	assert.Equal(t, lines.lines[0].Plain(nil), "This is a compressed file", "%s", filename)
}

func TestCompressedFiles(t *testing.T) {
	testCompressedFile(t, "compressed.txt.gz")
	testCompressedFile(t, "compressed.txt.bz2")
	testCompressedFile(t, "compressed.txt.xz")
	testCompressedFile(t, "compressed.txt.zst")
	testCompressedFile(t, "compressed.txt.zstd")
}

func TestReadFileDoneNoHighlighting(t *testing.T) {
	testMe, err := NewReaderFromFilename(samplesDir+"/empty",
		*styles.Get("Native"), formatters.TTY, nil)
	assert.NilError(t, err)

	assert.NilError(t, testMe._wait())
}

func TestReadFileDoneYesHighlighting(t *testing.T) {
	testMe, err := NewReaderFromFilename("reader_test.go",
		*styles.Get("Native"), formatters.TTY, nil)
	assert.NilError(t, err)

	assert.NilError(t, testMe._wait())
}

func TestReadStreamDoneNoHighlighting(t *testing.T) {
	testMe := NewReaderFromStream("", strings.NewReader("Johan"), chroma.Style{}, nil, nil)

	assert.NilError(t, testMe._wait())
}

func TestReadStreamDoneYesHighlighting(t *testing.T) {
	testMe := NewReaderFromStream("",
		strings.NewReader("Johan"),
		*styles.Get("Native"), formatters.TTY, lexers.EmacsLisp)

	assert.NilError(t, testMe._wait())
}

func TestReadTextDone(t *testing.T) {
	testMe := NewReaderFromText("", "Johan")

	assert.NilError(t, testMe._wait())
}

// If people keep appending to the currently opened file we should show display
// those changes.
func TestReadUpdatingFile(t *testing.T) {
	// Make a temp file containing one line of text, ending with a newline
	file, err := os.CreateTemp("", "moar-TestReadUpdatingFile-*.txt")
	assert.NilError(t, err)
	defer os.Remove(file.Name())

	_, err = file.WriteString("First line\n")
	assert.NilError(t, err)

	// Start a reader on that file
	testMe, err := NewReaderFromFilename(file.Name(), *styles.Get("native"), formatters.TTY16m, nil)
	assert.NilError(t, err)

	// Wait for the reader to finish reading
	assert.NilError(t, testMe._wait())

	// Verify we got the single line
	allLines, _ := testMe.GetLines(linenumbers.LineNumber{}, 10)
	assert.Equal(t, len(allLines.lines), 1)
	assert.Equal(t, allLines.lines[0].Plain(nil), "First line")

	// Append a line to the file
	_, err = file.WriteString("Second line\n")
	assert.NilError(t, err)

	// Give the reader some time to react
	for i := 0; i < 20; i++ {
		allLines, _ = testMe.GetLines(linenumbers.LineNumber{}, 10)
		if len(allLines.lines) == 2 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Verify we got the two lines
	allLines, _ = testMe.GetLines(linenumbers.LineNumber{}, 10)
	assert.Equal(t, len(allLines.lines), 2, "Expected two lines after adding a second one, got %d", len(allLines.lines))
	assert.Equal(t, allLines.lines[0].Plain(nil), "First line")
	assert.Equal(t, allLines.lines[1].Plain(nil), "Second line")
}

// How long does it take to read a file?
//
// This can be slow due to highlighting.
//
// Run with: go test -run='^$' -bench=. . ./...
func BenchmarkReaderDone(b *testing.B) {
	filename := "pager.go" // This is our longest .go file
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		// This is our longest .go file
		readMe, err := NewReaderFromFilename(filename, *styles.Get("native"), formatters.TTY16m, nil)
		assert.NilError(b, err)

		assert.NilError(b, readMe._wait())
		assert.NilError(b, readMe.err)
	}
}

// Try loading a large file
func BenchmarkReadLargeFile(b *testing.B) {
	// Try loading a file this large
	const largeSizeBytes = 35_000_000

	// First, create it from something...
	inputFilename := "pager.go"
	contents, err := os.ReadFile(inputFilename)
	assert.NilError(b, err)

	testdir := b.TempDir()
	largeFileName := testdir + "/large-file"
	largeFile, err := os.Create(largeFileName)
	assert.NilError(b, err)

	totalBytesWritten := 0
	for totalBytesWritten < largeSizeBytes {
		written, err := largeFile.Write(contents)
		assert.NilError(b, err)

		totalBytesWritten += written
	}
	err = largeFile.Close()
	assert.NilError(b, err)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		readMe, err := NewReaderFromFilename(largeFileName, *styles.Get("native"), formatters.TTY16m, nil)
		assert.NilError(b, err)

		assert.NilError(b, readMe._wait())
		assert.NilError(b, readMe.err)
	}
}

// Count lines in pager.go
func BenchmarkCountLines(b *testing.B) {
	// First, get some sample lines...
	inputFilename := "pager.go"
	contents, err := os.ReadFile(inputFilename)
	assert.NilError(b, err)

	testdir := b.TempDir()
	countFileName := testdir + "/count-file"
	countFile, err := os.Create(countFileName)
	assert.NilError(b, err)

	// 1000x makes this take about 12ms on my machine right now. Before 1000x
	// the numbers fluctuated much more.
	for n := 0; n < b.N*1000; n++ {
		_, err := countFile.Write(contents)
		assert.NilError(b, err)
	}
	err = countFile.Close()
	assert.NilError(b, err)

	b.ResetTimer()
	_, err = countLines(countFileName)
	assert.NilError(b, err)
}
