package m

import (
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

// NOTE: You can find related tests in pager_test.go.

func testHorizontalCropping(t *testing.T, contents string, firstVisibleColumn int, lastVisibleColumn int, expected string) {
	pager := NewPager(nil)
	pager.ShowLineNumbers = false

	pager.screen = twin.NewFakeScreen(1+lastVisibleColumn-firstVisibleColumn, 99)
	pager.leftColumnZeroBased = firstVisibleColumn
	pager.scrollPosition = newScrollPosition("testHorizontalCropping")

	lineContents := NewLine(contents)
	numberedLine := NumberedLine{
		number: linenumbers.LineNumber{},
		line:   &lineContents,
	}
	screenLine := pager.renderLine(&numberedLine, pager.scrollPosition.internalDontTouch)
	assert.Equal(t, rowToString(screenLine[0].cells), expected)
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

func TestCreateScreenLineChopWideCharLeft(t *testing.T) {
	testHorizontalCropping(t, "上午下", 0, 10, "上午下")
	testHorizontalCropping(t, "上午下", 1, 10, "<午下")
	testHorizontalCropping(t, "上午下", 2, 10, "< 下")
	testHorizontalCropping(t, "上午下", 3, 10, "<下")
	testHorizontalCropping(t, "上午下", 4, 10, "<")
	testHorizontalCropping(t, "上午下", 5, 10, "<")
	testHorizontalCropping(t, "上午下", 6, 10, "<")
	testHorizontalCropping(t, "上午下", 7, 10, "<")
}

func TestCreateScreenLineChopWideCharRight(t *testing.T) {
	testHorizontalCropping(t, "上午下", 0, 6, "上午下")
	testHorizontalCropping(t, "上午下", 0, 5, "上午下")
	testHorizontalCropping(t, "上午下", 0, 4, "上午>")
	testHorizontalCropping(t, "上午下", 0, 3, "上 >")
	testHorizontalCropping(t, "上午下", 0, 2, "上>")
	testHorizontalCropping(t, "上午下", 0, 1, " >")
}

func TestEmpty(t *testing.T) {
	pager := Pager{
		screen: twin.NewFakeScreen(99, 10),

		// No lines available
		reader: NewReaderFromText("test", ""),

		scrollPosition: newScrollPosition("TestEmpty"),
	}

	rendered, statusText := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 0)
	assert.Equal(t, "test: <empty>", statusText)
	assert.Assert(t, pager.lineNumber() == nil)
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

	numberedLine := NumberedLine{
		number: linenumbers.LineNumber{},
		line:   &line,
	}
	rendered := pager.renderLine(&numberedLine, pager.scrollPosition.internalDontTouch)
	assert.DeepEqual(t, []renderedLine{
		{
			inputLine: linenumbers.LineNumber{},
			wrapIndex: 0,
			cells: []twin.StyledRune{
				{Rune: 'x', Style: twin.StyleDefault},
				{Rune: '"', Style: twin.StyleDefault.WithAttr(twin.AttrReverse)},
				{Rune: '"', Style: twin.StyleDefault.WithAttr(twin.AttrReverse)},
				{Rune: 'x', Style: twin.StyleDefault},
			},
		},
	}, rendered,
		cmp.AllowUnexported(twin.Style{}),
		cmp.AllowUnexported(renderedLine{}),
		cmp.AllowUnexported(linenumbers.LineNumber{}),
	)
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
		scrollPosition: *scrollPositionFromLineNumber("TestOverflowDown", linenumbers.LineNumberFromOneBased(42)),
	}

	rendered, statusText := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, "test: 1 line  100%", statusText)
	assert.Assert(t, pager.lineNumber().IsZero())
	assert.Equal(t, pager.deltaScreenLines(), 0)
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

	rendered, statusText := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, "test: 1 line  100%", statusText)
	assert.Assert(t, pager.lineNumber().IsZero())
	assert.Equal(t, pager.deltaScreenLines(), 0)
}

func TestWrapping(t *testing.T) {
	reader := NewReaderFromText("",
		"first line\nline two will be wrapped\nhere's the last line")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 40)

	pager.WrapLongLines = true
	pager.ShowLineNumbers = false

	assert.NilError(t, reader._wait())

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

	rendered, _ := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 0)
}
