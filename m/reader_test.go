package m

import (
	"math"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/styles"
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
	if err != nil {
		panic(err)
	}
	if countLinesCount != uint64(wcLineCount) {
		t.Errorf("Got %d lines from wc -l, but %d lines from our countLines() function", wcLineCount, countLinesCount)
	}
}

func testGetLines(t *testing.T, reader *Reader) {
	lines, _ := reader.GetLines(1, 10)
	if len(lines.lines) > 10 {
		t.Errorf("Asked for 10 lines, got too many: %d", len(lines.lines))
	}

	if len(lines.lines) < 10 {
		// No good plan for how to test short files, more than just
		// querying them, which we just did
		return
	}

	// Test clipping at the end
	lines, _ = reader.GetLines(math.MaxInt32, 10)
	if len(lines.lines) != 10 {
		t.Errorf("Asked for 10 lines but got %d", len(lines.lines))
		return
	}

	startOfLastSection := lines.firstLineOneBased
	lines, _ = reader.GetLines(startOfLastSection, 10)
	if lines.firstLineOneBased != startOfLastSection {
		t.Errorf("Expected start line %d when asking for the last 10 lines, got %d",
			startOfLastSection, lines.firstLineOneBased)
		return
	}
	if len(lines.lines) != 10 {
		t.Errorf("Expected 10 lines when asking for the last 10 lines, got %d",
			len(lines.lines))
		return
	}

	lines, _ = reader.GetLines(startOfLastSection+1, 10)
	if lines.firstLineOneBased != startOfLastSection {
		t.Errorf("Expected start line %d when asking for the last+1 10 lines, got %d",
			startOfLastSection, lines.firstLineOneBased)
		return
	}
	if len(lines.lines) != 10 {
		t.Errorf("Expected 10 lines when asking for the last+1 10 lines, got %d",
			len(lines.lines))
		return
	}

	lines, _ = reader.GetLines(startOfLastSection-1, 10)
	if lines.firstLineOneBased != startOfLastSection-1 {
		t.Errorf("Expected start line %d when asking for the last-1 10 lines, got %d",
			startOfLastSection, lines.firstLineOneBased)
		return
	}
	if len(lines.lines) != 10 {
		t.Errorf("Expected 10 lines when asking for the last-1 10 lines, got %d",
			len(lines.lines))
		return
	}
}

func getTestFiles() []string {
	files, err := os.ReadDir(samplesDir)
	if err != nil {
		panic(err)
	}

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
	for _, file := range getTestFiles() {
		if strings.HasSuffix(file, ".xz") {
			_, err := exec.LookPath("xz")
			if err != nil {
				t.Log("Not testing xz compressed file, xz not found in $PATH: ", file)
				continue
			}
		}

		if strings.HasSuffix(file, ".bz2") {
			_, err := exec.LookPath("bzip2")
			if err != nil {
				t.Log("Not testing bzip2 compressed file, bzip2 not found in $PATH: ", file)
				continue
			}
		}

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

	// Load the unformatted file
	rawBytes, err := os.ReadFile(filenameWithPath)
	if err != nil {
		panic(err)
	}
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
	if err != nil {
		panic(err)
	}
	err = reader._wait()
	if err != nil {
		panic(err)
	}

	highlightedLinesCount := reader.GetLineCount()
	assert.Check(t, rawLinesCount == highlightedLinesCount)
}

func TestGetLongLine(t *testing.T) {
	file := "../sample-files/very-long-line.txt"
	reader, err := NewReaderFromFilename(file, *styles.Get("native"), formatters.TTY16m, nil)
	if err != nil {
		panic(err)
	}
	if err := reader._wait(); err != nil {
		panic(err)
	}

	lines, overflow := reader.GetLines(1, 5)
	assert.Equal(t, lines.firstLineOneBased, 1)
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

func testStatusText(t *testing.T, fromLine int, toLine int, totalLines int, expected string) {
	testMe := getReaderWithLineCount(totalLines)
	linesRequested := toLine - fromLine + 1
	lines, _ := testMe.GetLines(fromLine, linesRequested)
	statusText := lines.statusText
	assert.Equal(t, statusText, expected)
}

func TestStatusText(t *testing.T) {
	testStatusText(t, 1, 10, 20, "20 lines  50%")
	testStatusText(t, 1, 5, 5, "5 lines  100%")
	testStatusText(t, 998, 999, 1000, "1000 lines  99%")

	testStatusText(t, 0, 0, 0, "<empty>")
	testStatusText(t, 1, 1, 1, "1 line  100%")

	// Test with filename
	testMe, err := NewReaderFromFilename(samplesDir+"/empty", *styles.Get("native"), formatters.TTY16m, nil)
	if err != nil {
		panic(err)
	}
	if err := testMe._wait(); err != nil {
		panic(err)
	}

	line, overflow := testMe.GetLines(0, 0)
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
	if err := reader._wait(); err != nil {
		panic(err)
	}

	lines, _ := reader.GetLines(1, 5)
	assert.Equal(t, lines.lines[0].Plain(nil), "This is a compressed file", "%s", filename)
}

func TestCompressedFiles(t *testing.T) {
	testCompressedFile(t, "compressed.txt.gz")

	_, err := exec.LookPath("bzip2")
	if err == nil {
		testCompressedFile(t, "compressed.txt.bz2")
	} else {
		t.Log("WARNING: bzip2 not found in path, not testing automatic bzip2 decompression")
	}

	_, err = exec.LookPath("xz")
	if err == nil {
		testCompressedFile(t, "compressed.txt.xz")
	} else {
		t.Log("WARNING: xz not found in path, not testing automatic xz decompression")
	}
}

func TestFilterNotInstalled(t *testing.T) {
	t.Skip("FIXME: Test what happens if we try to use a filter that is not installed")
}

func TestFilterFailure(t *testing.T) {
	t.Skip("FIXME: Test what happens if the filter command fails because of bad command line options")
}

func TestFilterPermissionDenied(t *testing.T) {
	t.Skip("FIXME: Test what happens if the filter command fails because it can't access the requested file")
}

func TestFilterFileNotFound(t *testing.T) {
	// What happens if the filter cannot read its input file?
	NonExistentPath := "/does-not-exist"

	reader, err := newReaderFromCommand(NonExistentPath, "cat")

	// Creating should be fine, it's waiting for it to finish that should fail.
	// Feel free to re-evaluate in the future.
	assert.Check(t, err == nil)

	err = reader._wait()
	assert.Check(t, err != nil)

	assert.Check(t, strings.Contains(err.Error(), NonExistentPath), err.Error())
}

func TestFilterNotAFile(t *testing.T) {
	t.Skip("FIXME: Test what happens if the filter command fails because the target is not a file")
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
		if err != nil {
			panic(err)
		}

		// Wait for the reader to finish
		//revive:disable-next-line:empty-block
		for !readMe.done.Load() {
		}
		if readMe.err != nil {
			panic(readMe.err)
		}
	}
}

// Try loading a large file
func BenchmarkReadLargeFile(b *testing.B) {
	// Try loading a file this large
	const largeSizeBytes = 35_000_000

	// First, create it from something...
	inputFilename := "pager.go"
	contents, err := os.ReadFile(inputFilename)
	if err != nil {
		panic(err)
	}

	testdir := b.TempDir()
	largeFileName := testdir + "/large-file"
	largeFile, err := os.Create(largeFileName)
	if err != nil {
		panic(err)
	}

	totalBytesWritten := 0
	for totalBytesWritten < largeSizeBytes {
		written, err := largeFile.Write(contents)
		if err != nil {
			panic(err)
		}

		totalBytesWritten += written
	}
	err = largeFile.Close()
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		readMe, err := NewReaderFromFilename(largeFileName, *styles.Get("native"), formatters.TTY16m, nil)
		if err != nil {
			panic(err)
		}

		// Wait for the reader to finish
		for !readMe.done.Load() {
		}
		if readMe.err != nil {
			panic(readMe.err)
		}
	}
}

// Count lines in pager.go
func BenchmarkCountLines(b *testing.B) {
	// First, get some sample lines...
	inputFilename := "pager.go"
	contents, err := os.ReadFile(inputFilename)
	if err != nil {
		panic(err)
	}

	testdir := b.TempDir()
	countFileName := testdir + "/count-file"
	countFile, err := os.Create(countFileName)
	if err != nil {
		panic(err)
	}

	// 1000x makes this take about 12ms on my machine right now. Before 1000x
	// the numbers fluctuated much more.
	for n := 0; n < b.N*1000; n++ {
		_, err := countFile.Write(contents)
		if err != nil {
			panic(err)
		}
	}
	err = countFile.Close()
	if err != nil {
		panic(err)
	}

	b.ResetTimer()
	_, err = countLines(countFileName)
	if err != nil {
		panic(err)
	}
}
