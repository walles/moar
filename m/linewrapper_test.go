package m

import (
	"testing"

	"github.com/walles/moar/twin"
	"gotest.tools/assert"
)

func tokenize(input string) []twin.Cell {
	line := NewLine(input)
	return line.Tokens()
}

func TestEnoughRoomNoWrapping(t *testing.T) {
	toWrap := tokenize("This is a test")
	wrapped := wrapLine(20, toWrap)
	assert.Equal(t, wrapped, [][]twin.Cell{toWrap})
}

func TestSimpleWrap(t *testing.T) {
	toWrap := tokenize("This is a test")
	wrapped := wrapLine(12, toWrap)
	assert.Equal(t, wrapped, [][]twin.Cell{
		tokenize("This is a"),
		tokenize("test"),
	})
}

func TestWordLongerThanLine(t *testing.T) {
	toWrap := tokenize("intermediary")
	wrapped := wrapLine(6, toWrap)
	assert.Equal(t, wrapped, [][]twin.Cell{
		tokenize("interm"),
		tokenize("ediary"),
	})
}
