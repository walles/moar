package m

import (
	"strings"
	"testing"
)

func TestEmpty(t *testing.T) {
	reader := strings.NewReader("")
	lineReader := NewLineReader(reader)
	line, err := lineReader.GetLine()
	if err != nil {
		panic(err)
	}
	if line != nil {
		t.Errorf("Line should have been nil: <%s>", *line)
	}
}

func TestOneLineTrailingNewline(t *testing.T) {
	reader := strings.NewReader("apor\n")
	lineReader := NewLineReader(reader)

	line, err := lineReader.GetLine()
	if err != nil {
		panic(err)
	}
	if *line != "apor" {
		t.Errorf("Unexpected line contents: <%s>", *line)
	}

	line, err = lineReader.GetLine()
	if err != nil {
		panic(err)
	}
	if line != nil {
		t.Errorf("Second line should have been nil: <%s>", *line)
	}
}

func TestOneLineWithoutTrailingNewline(t *testing.T) {
	reader := strings.NewReader("apor")
	lineReader := NewLineReader(reader)

	line, err := lineReader.GetLine()
	if err != nil {
		panic(err)
	}
	if *line != "apor" {
		t.Errorf("Unexpected line contents: <%s>", *line)
	}

	line, err = lineReader.GetLine()
	if err != nil {
		panic(err)
	}
	if line != nil {
		t.Errorf("Second line should have been nil: <%s>", *line)
	}
}

// FIXME: Add test for Windows newlines
