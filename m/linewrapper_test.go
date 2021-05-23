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

func TestLeadingSpaceNoWrap(t *testing.T) {
	toWrap := tokenize(" abc")
	wrapped := wrapLine(20, toWrap)
	assertEqual(t, wrapped, [][]twin.Cell{
		tokenize(" abc"),
	})
}

func TestLeadingSpaceWithWrap(t *testing.T) {
	toWrap := tokenize(" abc")
	wrapped := wrapLine(2, toWrap)
	assertEqual(t, wrapped, [][]twin.Cell{
		tokenize(" a"),
		tokenize("bc"),
	})
}

func TestLeadingWrappedSpace(t *testing.T) {
	toWrap := tokenize("ab cd")
	wrapped := wrapLine(2, toWrap)
	assertEqual(t, wrapped, [][]twin.Cell{
		tokenize("ab"),
		tokenize("cd"),
	})
}

// FIXME: Test word wrapping

// FIXME: Test wrapping on single dashes
