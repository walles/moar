package m

import (
	"reflect"
	"testing"

	"github.com/walles/moar/twin"
	"gotest.tools/assert"
)

func tokenize(input string) []twin.Cell {
	line := NewLine(input)
	return line.HighlightedTokens(nil)
}

func TestEnoughRoomNoWrapping(t *testing.T) {
	toWrap := tokenize("This is a test")
	wrapped := wrapLine(20, toWrap)
	assert.Assert(t, reflect.DeepEqual(wrapped, [][]twin.Cell{toWrap}))
}

func TestWrapEmpty(t *testing.T) {
	empty := tokenize("")
	wrapped := wrapLine(20, empty)
	assert.Assert(t, reflect.DeepEqual(wrapped, [][]twin.Cell{empty}))
}

func TestWordLongerThanLine(t *testing.T) {
	toWrap := tokenize("intermediary")
	wrapped := wrapLine(6, toWrap)
	assert.Assert(t, reflect.DeepEqual(wrapped, [][]twin.Cell{
		tokenize("interm"),
		tokenize("ediary"),
	}))
}

// FIXME: Test word wrapping

// FIXME: Test wrapping with multiple consecutive spaces

// FIXME: Test wrapping on single dashes

// FIXME: Test wrapping with double dashes (not sure what we should do with those)

// FIXME: Test wrapping formatted strings, is there formatting that should affect the wrapping

// FIXME: Test wrapping with trailing whitespace
