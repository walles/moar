package m

import (
	"strings"
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
	assert.Equal(t, "test: <empty>", statusText)
	assert.Equal(t, firstScreenLine, 0)
}

func TestOverflowDown(t *testing.T) {
	pager := Pager{
		screen: twin.NewFakeScreen(
			10, // Longer than the raw line, we're testing vertical overflow, not horizontal
			2,  // Single line of contents + one status line
		),

		// Single line of input
		reader: NewReaderFromText("test", "hej"),

		// This value can be anything and should be clipped, that's what we're testing
		firstLineOneBased: 42,
	}

	rendered, statusText, firstScreenLine := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, "test: 1 line  100%", statusText)
	assert.Equal(t, firstScreenLine, 1)
}

func TestOverflowUp(t *testing.T) {
	pager := Pager{
		screen: twin.NewFakeScreen(
			10, // Longer than the raw line, we're testing vertical overflow, not horizontal
			2,  // Single line of contents + one status line
		),

		// Single line of input
		reader: NewReaderFromText("test", "hej"),

		firstLineOneBased: 1,
	}

	rendered, statusText, firstScreenLine := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, "test: 1 line  100%", statusText)
	assert.Equal(t, firstScreenLine, 1)
}

func TestWrapping(t *testing.T) {
	reader := NewReaderFromStream("",
		strings.NewReader("first line\nline two will be wrapped\nhere's the last line"))
	pager := NewPager(reader)
	pager.WrapLongLines = true
	pager.ShowLineNumbers = false

	// Wait for reader to finish reading
	<-reader.done

	// This is what we're testing really
	pager._ScrollToEnd()

	// Higher than needed, we'll just be validating the necessary lines at the
	// top.
	screen := twin.NewFakeScreen(10, 99)

	// Exit immediately
	pager.Quit()

	// Get contents onto our fake screen
	pager.StartPaging(screen)
	pager._Redraw("")

	actual := strings.Join([]string{
		rowToString(screen.GetRow(0)),
		rowToString(screen.GetRow(1)),
		rowToString(screen.GetRow(2)),
		rowToString(screen.GetRow(3)),
		rowToString(screen.GetRow(4)),
		rowToString(screen.GetRow(5)),
		rowToString(screen.GetRow(6)),
		rowToString(screen.GetRow(7)),
	}, "\n")
	assert.Equal(t, actual, strings.Join([]string{
		"first line",
		"line two",
		"will be",
		"wrapped",
		"here's the",
		"last line",
		"---",
		"",
	}, "\n"))
}
