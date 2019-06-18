package m

import (
	"io/ioutil"
	"math"
	"path"
	"runtime"
	"testing"
)

func _TestGetLines(t *testing.T, reader *Reader) {
	t.Logf("Testing file: %s...", reader.name)

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

func _GetTestFiles() []string {
	// From: https://coderwall.com/p/_fmbug/go-get-path-to-current-file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Getting current filename failed")
	}

	samplesDir := path.Join(path.Dir(filename), "../sample-files")

	files, err := ioutil.ReadDir(samplesDir)
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

// FIXME: Add test for code highlighting using http://www.andre-simon.de/zip/download.php
