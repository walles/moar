package m

import (
	"testing"

	"github.com/walles/moar/twin"
	"gotest.tools/assert"
)

func testHorizontalCropping(t *testing.T, contents string, firstIndex int, lastIndex int, expected string) {
	pager := Pager{
		screen:              twin.NewFakeScreen(1+lastIndex-firstIndex, 99),
		leftColumnZeroBased: firstIndex,
	}
	lineContents := NewLine(contents).HighlightedTokens(nil)
	screenLine := pager.createScreenLine(nil, 0, lineContents)
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
	pager := Pager{
		screen: twin.NewFakeScreen(99, 10),

		// No lines available
		reader: NewReaderFromText("test", ""),
	}

	rendered, statusText, firstScreenLine := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 0)
	assert.Equal(t, "johan", statusText)
	assert.Equal(t, firstScreenLine, 0)
}

func TestOverflowDown(t *testing.T) {
	pager := Pager{
		screen: twin.NewFakeScreen(
			10, // Longer than the raw line, we're testing vertical overflow, not horizontal
			1,  // Single line screen
		),

		// Single line of input
		reader: NewReaderFromText("test", "hej"),

		// This value can be anything and should be clipped, that's what we're testing
		firstLineOneBased: 42,
	}

	rendered, statusText, firstScreenLine := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, "johan", statusText)
	assert.Equal(t, firstScreenLine, 1)
}

func TestOverflowUp(t *testing.T) {
	pager := Pager{
		screen: twin.NewFakeScreen(
			10, // Longer than the raw line, we're testing vertical overflow, not horizontal
			1,  // Single line screen
		),

		// Single line of input
		reader: NewReaderFromText("test", "hej"),

		firstLineOneBased: 1,
	}

	rendered, statusText, firstScreenLine := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, "johan", statusText)
	assert.Equal(t, firstScreenLine, 1)
}
