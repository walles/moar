package m

import (
	"io/ioutil"
	"math"
	"path"
	"runtime"
	"strings"
	"testing"

	"gotest.tools/assert"
)

func _TestGetLines(t *testing.T, reader *Reader) {
	t.Logf("Testing file: %s...", *reader.name)

	lines := reader.GetLines(1, 10)
	if len(lines.lines) > 10 {
		t.Errorf("Asked for 10 lines, got too many: %d", len(lines.lines))
	}

	if len(lines.lines) < 10 {
		// No good plan for how to test short files more
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
		reader, e := NewReaderFromFilename(file)
		if e != nil {
			t.Errorf("Error opening file <%s>: %s", file, e.Error())
			continue
		}

		_TestGetLines(t, reader)
	}
}

func _GetReaderWithLineCount(totalLines int) *Reader {
	reader, err := NewReaderFromStream(strings.NewReader(strings.Repeat("x\n", totalLines)))
	if err != nil {
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
	statusText := testMe.GetLines(0, 0).statusText
	assert.Equal(t, statusText, "null: <empty>")
}

func _TestCompressedFile(t *testing.T, filename string) {
	reader, e := NewReaderFromFilename(_GetSamplesDir() + "/" + filename)
	if e != nil {
		t.Errorf("Error opening file <%s>: %s", filename, e.Error())
		panic(e)
	}

	assert.Equal(t, reader.GetLines(1, 5).lines, [...]string{"This is a compressed file"})
}

func TestCompressedFiles(t *testing.T) {
	_TestCompressedFile(t, "compressed.gz")
	_TestCompressedFile(t, "compressed.bz2")
	_TestCompressedFile(t, "compressed.xz")
}
