package m

import (
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

//revive:disable:empty-block

func testHorizontalCropping(t *testing.T, contents string, firstIndex int, lastIndex int, expected string, expectedOverflow overflowState) {
	pager := NewPager(nil)
	pager.ShowLineNumbers = false

	pager.screen = twin.NewFakeScreen(1+lastIndex-firstIndex, 99)
	pager.leftColumnZeroBased = firstIndex
	pager.scrollPosition = newScrollPosition("testHorizontalCropping")

	lineContents := NewLine(contents)
	screenLine, didOverflow := pager.renderLine(&lineContents, 0, pager.scrollPosition.internalDontTouch)
	assert.Equal(t, rowToString(screenLine[0].cells), expected)
	assert.Equal(t, didOverflow, expectedOverflow)
}

func TestCreateScreenLine(t *testing.T) {
	testHorizontalCropping(t, "abc", 0, 10, "abc", didFit)
}

func TestCreateScreenLineCanScrollLeft(t *testing.T) {
	testHorizontalCropping(t, "abc", 1, 10, "<c", didOverflow)
}

func TestCreateScreenLineCanScrollRight(t *testing.T) {
	testHorizontalCropping(t, "abc", 0, 1, "a>", didOverflow)
}

func TestCreateScreenLineCanAlmostScrollRight(t *testing.T) {
	testHorizontalCropping(t, "abc", 0, 2, "abc", didFit)
}

func TestCreateScreenLineCanScrollBoth(t *testing.T) {
	testHorizontalCropping(t, "abcde", 1, 3, "<c>", didOverflow)
}

func TestCreateScreenLineCanAlmostScrollBoth(t *testing.T) {
	testHorizontalCropping(t, "abcd", 1, 3, "<cd", didOverflow)
}

func TestEmpty(t *testing.T) {
	pager := Pager{
		screen: twin.NewFakeScreen(99, 10),

		// No lines available
		reader: NewReaderFromText("test", ""),

		scrollPosition: newScrollPosition("TestEmpty"),
	}

	rendered, statusText, overflow := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 0)
	assert.Equal(t, "test: <empty>", statusText)
	assert.Equal(t, pager.lineNumberOneBased(), 0)
	assert.Equal(t, overflow, didFit)
}

// Repro case for a search bug discovered in v1.9.8.
func TestSearchHighlight(t *testing.T) {
	line := Line{
		raw: "x\"\"x",
	}
	pager := Pager{
		screen:        twin.NewFakeScreen(100, 10),
		searchPattern: regexp.MustCompile("\""),
	}

	rendered, overflow := pager.renderLine(&line, 1, pager.scrollPosition.internalDontTouch)
	assert.DeepEqual(t, []renderedLine{
		{
			inputLineOneBased: 1,
			wrapIndex:         0,
			cells: []twin.Cell{
				{Rune: 'x', Style: twin.StyleDefault},
				{Rune: '"', Style: twin.StyleDefault.WithAttr(twin.AttrReverse)},
				{Rune: '"', Style: twin.StyleDefault.WithAttr(twin.AttrReverse)},
				{Rune: 'x', Style: twin.StyleDefault},
			},
		},
	}, rendered, cmp.AllowUnexported(twin.Style{}), cmp.AllowUnexported(renderedLine{}))
	assert.Equal(t, overflow, didFit)
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
		scrollPosition: *scrollPositionFromLineNumber("TestOverflowDown", 42),
	}

	rendered, statusText, overflow := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, "test: 1 line  100%", statusText)
	assert.Equal(t, pager.lineNumberOneBased(), 1)
	assert.Equal(t, pager.deltaScreenLines(), 0)
	assert.Equal(t, overflow, didFit)
}

func TestOverflowUp(t *testing.T) {
	pager := Pager{
		screen: twin.NewFakeScreen(
			10, // Longer than the raw line, we're testing vertical overflow, not horizontal
			2,  // Single line of contents + one status line
		),

		// Single line of input
		reader: NewReaderFromText("test", "hej"),

		// NOTE: scrollPosition intentionally not initialized
	}

	rendered, statusText, overflow := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, "test: 1 line  100%", statusText)
	assert.Equal(t, pager.lineNumberOneBased(), 1)
	assert.Equal(t, pager.deltaScreenLines(), 0)
	assert.Equal(t, overflow, didFit)
}

func TestWrapping(t *testing.T) {
	reader := NewReaderFromStream("",
		strings.NewReader("first line\nline two will be wrapped\nhere's the last line"))
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 40)

	pager.WrapLongLines = true
	pager.ShowLineNumbers = false

	// Wait for reader to finish reading
	for !reader.done.Load() {
	}

	// This is what we're testing really
	pager.scrollToEnd()

	// Higher than needed, we'll just be validating the necessary lines at the
	// top.
	screen := twin.NewFakeScreen(10, 99)

	// Exit immediately
	pager.Quit()

	// Get contents onto our fake screen
	pager.StartPaging(screen, nil, nil)
	pager.redraw("")

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

// Repro for https://github.com/walles/moar/issues/153
func TestOneLineTerminal(t *testing.T) {
	pager := Pager{
		// Single line terminal window, this is what we're testing
		screen: twin.NewFakeScreen(20, 1),

		reader:        NewReaderFromText("test", "hej"),
		ShowStatusBar: true,
	}

	rendered, _, _ := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 0)
}
