package m

import (
	"testing"

	"gotest.tools/assert"
)

func testHorizontalCropping(t *testing.T, contents string, firstIndex int, lastIndex int, expected string) {
	screenLines := ScreenLines{width: 1 + lastIndex - firstIndex, leftColumnZeroBased: firstIndex}
	lineContents := NewLine(contents).HighlightedTokens(nil)
	screenLine := screenLines.createScreenLine(nil, 0, lineContents)
	assert.Equal(t, rowToString(screenLine), expected)
}

func TestCreateScreenLine(t *testing.T) {
	testHorizontalCropping(t, "abc", 0, 10, "abc")
}

func TestCreateScreenLineCanScrollLeft(t *testing.T) {
	testHorizontalCropping(t, "abc", 1, 10, "<c")
}

func TestCreateScreenLineCanScrollRight(t *testing.T) {
	testHorizontalCropping(t, "abc", 0, 1, "a>")
}

func TestCreateScreenLineCanAlmostScrollRight(t *testing.T) {
	testHorizontalCropping(t, "abc", 0, 2, "abc")
}

func TestCreateScreenLineCanScrollBoth(t *testing.T) {
	testHorizontalCropping(t, "abcde", 1, 3, "<c>")
}

func TestCreateScreenLineCanAlmostScrollBoth(t *testing.T) {
	testHorizontalCropping(t, "abcd", 1, 3, "<cd")
}

func TestEmpty(t *testing.T) {
	// This is what _GetLinesUnlocked() returns on no-lines-available
	inputLines := InputLines{
		lines:             nil,
		firstLineOneBased: 0,
	}

	screenLines := ScreenLines{
		inputLines: &inputLines,
		height:     10,
	}

	rendered, firstScreenLine := screenLines.renderScreenLines()
	assert.Equal(t, len(rendered), 0)
	assert.Equal(t, firstScreenLine, 0)
}

func TestOverflowDown(t *testing.T) {
	// Set up a single line input
	line := Line{
		raw: "hej",
	}
	inputLines := InputLines{
		lines:             []*Line{&line},
		firstLineOneBased: 1,
	}

	// Set up a single line screen
	screenLines := ScreenLines{
		inputLines: &inputLines,
		height:     1,
		width:      10, // Longer than the raw line, we're testing vertical overflow, not horizontal

		// This value can be anything and should be clipped, that's what we're testing
		firstInputLineOneBased: 42,
	}

	rendered, firstScreenLine := screenLines.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, firstScreenLine, 1)
}

func TestOverflowUp(t *testing.T) {
	// Set up a single line input
	line := Line{
		raw: "hej",
	}
	inputLines := InputLines{
		lines:             []*Line{&line},
		firstLineOneBased: 1,
	}

	// Set up a single line screen
	screenLines := ScreenLines{
		inputLines: &inputLines,
		height:     1,
		width:      10, // Longer than the raw line, we're testing vertical overflow, not horizontal

		// This value can be anything and should be clipped, that's what we're testing
		firstInputLineOneBased: 0,
	}

	rendered, firstScreenLine := screenLines.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, firstScreenLine, 1)
}
