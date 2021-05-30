package m

import (
	"testing"

	"gotest.tools/assert"
)

func TestCreateScreenLine(t *testing.T) {
	screenLines := ScreenLines{width: 10}
	lineContents := NewLine("abc").HighlightedTokens(nil)
	screenLine := screenLines.createScreenLine(nil, 0, lineContents)
	assert.Equal(t, rowToString(screenLine), "abc")
}

func TestCreateScreenLineCanScrollLeft(t *testing.T) {
	screenLines := ScreenLines{width: 10, leftColumnZeroBased: 1}
	lineContents := NewLine("abc").HighlightedTokens(nil)
	screenLine := screenLines.createScreenLine(nil, 0, lineContents)
	assert.Equal(t, rowToString(screenLine), "<c")
}

func TestCreateScreenLineCanScrollRight(t *testing.T) {
	screenLines := ScreenLines{width: 2}
	lineContents := NewLine("abc").HighlightedTokens(nil)
	screenLine := screenLines.createScreenLine(nil, 0, lineContents)
	assert.Equal(t, rowToString(screenLine), "a>")
}

func TestCreateScreenLineCanAlmostScrollRight(t *testing.T) {
	screenLines := ScreenLines{width: 3}
	lineContents := NewLine("abc").HighlightedTokens(nil)
	screenLine := screenLines.createScreenLine(nil, 0, lineContents)
	assert.Equal(t, rowToString(screenLine), "abc")
}
