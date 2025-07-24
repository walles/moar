package m

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/walles/moar/m/linemetadata"
	"github.com/walles/moar/m/reader"
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

	lineContents := reader.NewLine(contents)
	numberedLine := reader.NumberedLine{
		Line: &lineContents,
	}
	screenLine := pager.renderLine(&numberedLine, pager.getLineNumberPrefixLength(numberedLine.Number))
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
		reader: reader.NewFromText("test", ""),

		scrollPosition: newScrollPosition("TestEmpty"),
	}
	pager.filteringReader = FilteringReader{
		BackingReader: pager.reader,
		FilterPattern: &pager.filterPattern,
	}

	rendered, statusText := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 0)
	assert.Equal(t, "test: <empty>", statusText)
	assert.Assert(t, pager.lineIndex() == nil)
}

// Repro case for a search bug discovered in v1.9.8.
func TestSearchHighlight(t *testing.T) {
	line := reader.NewLine("x\"\"x")
	pager := Pager{
		screen:        twin.NewFakeScreen(100, 10),
		searchPattern: regexp.MustCompile("\""),
	}
	pager.filteringReader = FilteringReader{
		BackingReader: pager.reader,
		FilterPattern: &pager.filterPattern,
	}

	numberedLine := reader.NumberedLine{
		Line: &line,
	}
	rendered := pager.renderLine(&numberedLine, pager.getLineNumberPrefixLength(numberedLine.Number))
	assert.DeepEqual(t, []renderedLine{
		{
			inputLineIndex: linemetadata.Index{},
			wrapIndex:      0,
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
		cmp.AllowUnexported(linemetadata.Number{}),
		cmp.AllowUnexported(linemetadata.Index{}),
	)
}

func TestOverflowDown(t *testing.T) {
	pager := Pager{
		screen: twin.NewFakeScreen(
			10, // Longer than the raw line, we're testing vertical overflow, not horizontal
			2,  // Single line of contents + one status line
		),

		// Single line of input
		reader: reader.NewFromText("test", "hej"),

		// This value can be anything and should be clipped, that's what we're testing
		scrollPosition: *scrollPositionFromIndex("TestOverflowDown", linemetadata.IndexFromOneBased(42)),
	}
	pager.filteringReader = FilteringReader{
		BackingReader: pager.reader,
		FilterPattern: &pager.filterPattern,
	}

	rendered, statusText := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, "test: 1 line  100%", statusText)
	assert.Assert(t, pager.lineIndex().IsZero())
	assert.Equal(t, pager.deltaScreenLines(), 0)
}

func TestOverflowUp(t *testing.T) {
	pager := Pager{
		screen: twin.NewFakeScreen(
			10, // Longer than the raw line, we're testing vertical overflow, not horizontal
			2,  // Single line of contents + one status line
		),

		// Single line of input
		reader: reader.NewFromText("test", "hej"),

		// NOTE: scrollPosition intentionally not initialized
	}
	pager.filteringReader = FilteringReader{
		BackingReader: pager.reader,
		FilterPattern: &pager.filterPattern,
	}

	rendered, statusText := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1)
	assert.Equal(t, "hej", rowToString(rendered[0]))
	assert.Equal(t, "test: 1 line  100%", statusText)
	assert.Assert(t, pager.lineIndex().IsZero())
	assert.Equal(t, pager.deltaScreenLines(), 0)
}

func TestWrapping(t *testing.T) {
	reader := reader.NewFromText("",
		"first line\nline two will be wrapped\nhere's the last line")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 40)

	pager.WrapLongLines = true
	pager.ShowLineNumbers = false

	assert.NilError(t, reader.Wait())

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

		reader:        reader.NewFromText("test", "hej"),
		ShowStatusBar: true,
	}
	pager.filteringReader = FilteringReader{
		BackingReader: pager.reader,
		FilterPattern: &pager.filterPattern,
	}

	rendered, _ := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 0)
}

// What happens if we are scrolled to the bottom of a 1000 lines file, and then
// add a filter matching only the first line?
//
// What should happen is that we should go as far down as possible.
func TestShortenedInput(t *testing.T) {
	pager := Pager{
		screen: twin.NewFakeScreen(20, 10),

		// 1000 lines of input, we will scroll to the bottom
		reader: reader.NewFromText("test", "first\n"+strings.Repeat("line\n", 1000)),

		scrollPosition: newScrollPosition("TestShortenedInput"),
	}
	pager.filteringReader = FilteringReader{
		BackingReader: pager.reader,
		FilterPattern: &pager.filterPattern,
	}

	pager.scrollToEnd()
	assert.Equal(t, pager.lineIndex().Index(), 991, "This should have been the effect of calling scrollToEnd()")

	pager.mode = &PagerModeFilter{pager: &pager}
	pager.filterPattern = regexp.MustCompile("first") // Match only the first line

	rendered, _ := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 1, "Should have rendered one line")
	assert.Equal(t, "first", rowToString(rendered[0]))
	assert.Equal(t, pager.lineIndex().Index(), 0, "Should have scrolled to the first line")
}

// - Start with a 1000 lines file
// - Scroll to the bottom
// - Add a filter matching the first 100 lines
// - Render
// - Verify that the 10 last matching lines were rendered
func TestShortenedInputManyLines(t *testing.T) {
	lines := []string{"first"}
	for i := range 999 {
		if i < 100 {
			lines = append(lines, "match "+strconv.Itoa(i))
		} else {
			lines = append(lines, "other "+strconv.Itoa(i))
		}
	}

	pager := Pager{
		screen:         twin.NewFakeScreen(20, 10),
		reader:         reader.NewFromText("test", strings.Join(lines, "\n")),
		scrollPosition: newScrollPosition("TestShortenedInputManyLines"),
	}
	pager.filteringReader = FilteringReader{
		BackingReader: pager.reader,
		FilterPattern: &pager.filterPattern,
	}

	pager.scrollToEnd()
	assert.Equal(t, pager.lineIndex().Index(), 990, "Should be at the last line before filtering")

	pager.mode = &PagerModeFilter{pager: &pager}
	pager.filterPattern = regexp.MustCompile(`^match`)

	rendered, _ := pager.renderScreenLines()
	assert.Equal(t, len(rendered), 10, "Should have rendered 10 lines")

	expectedLines := []string{}
	for i := 90; i < 100; i++ {
		expectedLines = append(expectedLines, "match "+strconv.Itoa(i))
	}
	for i, row := range rendered {
		assert.Equal(t, rowToString(row), expectedLines[i], "Line %d mismatch", i)
	}
	assert.Equal(t, pager.lineIndex().Index(), 90, "The last lines should now be visible")
	assert.Equal(t, "match 99", rowToString(rendered[len(rendered)-1]))
}
