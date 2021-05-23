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

func TestEnoughRoomNoWrapping(t *testing.T) {
	toWrap := tokenize("This is a test")
	wrapped := wrapLine(20, toWrap)
	assertEqual(t, wrapped, [][]twin.Cell{toWrap})
}

func TestWrapEmpty(t *testing.T) {
	empty := tokenize("")
	wrapped := wrapLine(20, empty)
	assertEqual(t, wrapped, [][]twin.Cell{empty})
}

func TestWordLongerThanLine(t *testing.T) {
	toWrap := tokenize("intermediary")
	wrapped := wrapLine(6, toWrap)
	assertEqual(t, wrapped, [][]twin.Cell{
		tokenize("interm"),
		tokenize("ediary"),
	})
}

// FIXME: Test word wrapping

// FIXME: Test wrapping with multiple consecutive spaces

// FIXME: Test wrapping on single dashes

// FIXME: Test wrapping with double dashes (not sure what we should do with those)

// FIXME: Test wrapping formatted strings, is there formatting that should affect the wrapping

// FIXME: Test wrapping with trailing whitespace
