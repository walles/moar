package m

import (
	"testing"

	"gotest.tools/assert"
)

func testCropping(t *testing.T, contents string, firstIndex int, lastIndex int, expected string) {
	screenLines := ScreenLines{width: 1 + lastIndex - firstIndex, leftColumnZeroBased: firstIndex}
	lineContents := NewLine(contents).HighlightedTokens(nil)
	screenLine := screenLines.createScreenLine(nil, 0, lineContents)
	assert.Equal(t, rowToString(screenLine), expected)
}

func TestCreateScreenLine(t *testing.T) {
	testCropping(t, "abc", 0, 10, "abc")
}

func TestCreateScreenLineCanScrollLeft(t *testing.T) {
	testCropping(t, "abc", 1, 10, "<c")
}

func TestCreateScreenLineCanScrollRight(t *testing.T) {
	testCropping(t, "abc", 0, 1, "a>")
}

func TestCreateScreenLineCanAlmostScrollRight(t *testing.T) {
	testCropping(t, "abc", 0, 2, "abc")
}

func TestCreateScreenLineCanScrollBoth(t *testing.T) {
	testCropping(t, "abcde", 1, 3, "<c>")
}

func TestCreateScreenLineCanAlmostScrollBoth(t *testing.T) {
	testCropping(t, "abcd", 1, 3, "<cd")
}
