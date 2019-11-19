package m

import (
	"io/ioutil"
	"math"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func _TestGetLineCount(t *testing.T, reader *Reader) {
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
	fileLineCount, err := strconv.Atoi(wcNumberString)
	if err != nil {
		t.Error("Error counting lines of", *reader.name, err)
	}

	if strings.HasSuffix(*reader.name, "/line-without-newline.txt") {
		// "wc -l" thinks this file contains zero lines
		fileLineCount = 1
	}

	if reader.GetLineCount() != fileLineCount {
		t.Errorf("Got %d lines but expected %d: <%s>",
			reader.GetLineCount(), fileLineCount, *reader.name)
	}
}

func _TestGetLines(t *testing.T, reader *Reader) {
	t.Logf("Testing file: %s...", *reader.name)

	lines := reader.GetLines(1, 10)
	if len(lines.lines) > 10 {
		t.Errorf("Asked for 10 lines, got too many: %d", len(lines.lines))
	}

	if len(lines.lines) < 10 {
		// No good plan for how to test short files, more than just
		// querying them, which we just did
		return
	}

	// Test clipping at the end
	lines = reader.GetLines(math.MaxInt32, 10)
	if len(lines.lines) != 10 {
		t.Errorf("Asked for 10 lines but got %d", len(lines.lines))
		return
	}

	startOfLastSection := lines.firstLineOneBased
	lines = reader.GetLines(startOfLastSection, 10)
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

	lines = reader.GetLines(startOfLastSection+1, 10)
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

	lines = reader.GetLines(startOfLastSection-1, 10)
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

func _GetSamplesDir() string {
	// From: https://coderwall.com/p/_fmbug/go-get-path-to-current-file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Getting current filename failed")
	}

	return path.Join(path.Dir(filename), "../sample-files")
}

func _GetTestFiles() []string {
	files, err := ioutil.ReadDir(_GetSamplesDir())
	if err != nil {
		panic(err)
	}

	var filenames []string
	for _, file := range files {
		filenames = append(filenames, "../sample-files/"+file.Name())
	}

	return filenames
}

func TestGetLines(t *testing.T) {
	for _, file := range _GetTestFiles() {
		reader, err := NewReaderFromFilename(file)
		if err != nil {
			t.Errorf("Error opening file <%s>: %s", file, err.Error())
			continue
		}
		if err := reader._Wait(); err != nil {
			t.Errorf("Error reading file <%s>: %s", file, err.Error())
			continue
		}

		_TestGetLines(t, reader)
		_TestGetLineCount(t, reader)
	}
}

func _GetReaderWithLineCount(totalLines int) *Reader {
	reader := NewReaderFromStream(strings.NewReader(strings.Repeat("x\n", totalLines)), nil)
	if err := reader._Wait(); err != nil {
		panic(err)
	}

	return reader
}

func _TestStatusText(t *testing.T, fromLine int, toLine int, totalLines int, expected string) {
	testMe := _GetReaderWithLineCount(totalLines)
	linesRequested := toLine - fromLine + 1
	statusText := testMe.GetLines(fromLine, linesRequested).statusText
	assert.Equal(t, statusText, expected)
}

func TestStatusText(t *testing.T) {
	_TestStatusText(t, 1, 10, 20, "1-10/20 50%")
	_TestStatusText(t, 1, 5, 5, "1-5/5 100%")
	_TestStatusText(t, 998, 999, 1000, "998-999/1000 99%")

	_TestStatusText(t, 0, 0, 0, "<empty>")
	_TestStatusText(t, 1, 1, 1, "1-1/1 100%")

	// Test with filename
	testMe, err := NewReaderFromFilename("/dev/null")
	if err != nil {
		panic(err)
	}
	if err := testMe._Wait(); err != nil {
		panic(err)
	}

	statusText := testMe.GetLines(0, 0).statusText
	assert.Equal(t, statusText, "null: <empty>")
}

func _TestCompressedFile(t *testing.T, filename string) {
	filenameWithPath := _GetSamplesDir() + "/" + filename
	reader, e := NewReaderFromFilename(filenameWithPath)
	if e != nil {
		t.Errorf("Error opening file <%s>: %s", filenameWithPath, e.Error())
		panic(e)
	}
	if err := reader._Wait(); err != nil {
		panic(err)
	}

	assert.Equal(t, reader.GetLines(1, 5).lines[0], "This is a compressed file", "%s", filename)
}

func TestCompressedFiles(t *testing.T) {
	_TestCompressedFile(t, "compressed.txt.gz")
	_TestCompressedFile(t, "compressed.txt.bz2")
	_TestCompressedFile(t, "compressed.txt.xz")
}

func TestFilterNotInstalled(t *testing.T) {
	// FIXME: Test what happens if we try to use a filter that is not installed
}

func TestFilterFailure(t *testing.T) {
	// FIXME: Test what happens if the filter command fails because of bad command line options
}

func TestFilterPermissionDenied(t *testing.T) {
	// FIXME: Test what happens if the filter command fails because it can't access the requested file
}

func TestFilterFileNotFound(t *testing.T) {
	// The example file name needs to be one supported by one of our filters.
	// ".md" is good, it is supported by highlight.
	_, err := NewReaderFromFilename("/doesntexist.md")
	assert.Check(t, err != nil, "Opening non-existing file should have been an error")
	assert.Check(t, strings.HasPrefix(err.Error(), "open /doesntexist.md: "), err.Error())
}

func TestFilterNotAFile(t *testing.T) {
	// FIXME: Test what happens if the filter command fails because the target is not a file
}
