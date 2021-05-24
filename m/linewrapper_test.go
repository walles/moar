package m

import (
	"reflect"
	"testing"

	"github.com/walles/moar/twin"
)

func tokenize(input string) []twin.Cell {
	line := NewLine(input)
	return line.HighlightedTokens(nil)
}

func toString(cellLines [][]twin.Cell) string {
	returnMe := ""
	for _, cellLine := range cellLines {
		lineString := ""
		for _, cell := range cellLine {
			lineString += string(cell.Rune)
		}

		if len(returnMe) > 0 {
			returnMe += "\n"
		}
		returnMe += "<" + lineString + ">"
	}

	return returnMe
}

func assertEqual(t *testing.T, a [][]twin.Cell, b [][]twin.Cell) {
	if reflect.DeepEqual(a, b) {
		return
	}
	t.Errorf("Expected equal:\n%s\n\n%s", toString(a), toString(b))
}

func assertWrap(t *testing.T, input string, width int, wrappedLines ...string) {
	toWrap := tokenize(input)
	wrapped := wrapLine(width, toWrap)

	expected := [][]twin.Cell{}
	for _, wrappedLine := range wrappedLines {
		expected = append(expected, tokenize(wrappedLine))
	}

	assertEqual(t, wrapped, expected)
}

func TestEnoughRoomNoWrapping(t *testing.T) {
	assertWrap(t, "This is a test", 20, "This is a test")
}

func TestWrapEmpty(t *testing.T) {
	assertWrap(t, "", 20, "")
}

func TestWordLongerThanLine(t *testing.T) {
	assertWrap(t, "intermediary", 6, "interm", "ediary")
}

func TestLeadingSpaceNoWrap(t *testing.T) {
	assertWrap(t, " abc", 20, " abc")
}

func TestLeadingSpaceWithWrap(t *testing.T) {
	assertWrap(t, " abc", 2, " a", "bc")
}

func TestLeadingWrappedSpace(t *testing.T) {
	assertWrap(t, "ab cd", 2, "ab", "cd")
}

// FIXME: Test word wrapping

// FIXME: Test wrapping on single dashes
