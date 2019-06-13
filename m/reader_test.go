package m

import "testing"

func _TestGetLines(t *testing.T, reader *_Reader) {
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
	lineCount := reader.LineCount()
	lines = reader.GetLines(lineCount, 10)
	if len(lines.lines) != 10 {
		t.Errorf("Asked for 10 lines, got too few: %d", len(lines.lines))
		return
	}
	if lines.firstLineOneBased != lineCount - 9 {
		t.Errorf("Expected last line to be %d, was %d", lineCount - 9, lines.firstLineOneBased)
		return
	}
}

func TestGetLines(t *testing.T) {
	// FIXME: List all files in ../sample-files
	files := []string {"../sample-files/empty.txt"}
	for _, file := range files {
		reader, e := NewReaderFromFilename(file)
		if e != nil {
			t.Errorf("Error opening file <%s>: %s", file, e.Error())
			continue
		}

		_TestGetLines(t, reader)
	}
}