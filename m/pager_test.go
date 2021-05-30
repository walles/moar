package m

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/styles"
	"github.com/walles/moar/twin"
	"gotest.tools/assert"
)

func TestUnicodeRendering(t *testing.T) {
	reader := NewReaderFromStream("", strings.NewReader("åäö"))

	var answers = []twin.Cell{
		twin.NewCell('å', twin.StyleDefault),
		twin.NewCell('ä', twin.StyleDefault),
		twin.NewCell('ö', twin.StyleDefault),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		logDifference(t, expected, contents[pos])
	}
}

func logDifference(t *testing.T, expected twin.Cell, actual twin.Cell) {
	if actual.Rune == expected.Rune && actual.Style == expected.Style {
		return
	}

	t.Errorf("Expected %v, got %v", expected, actual)
}

func TestFgColorRendering(t *testing.T) {
	reader := NewReaderFromStream("", strings.NewReader(
		"\x1b[30ma\x1b[31mb\x1b[32mc\x1b[33md\x1b[34me\x1b[35mf\x1b[36mg\x1b[37mh\x1b[0mi"))

	var answers = []twin.Cell{
		twin.NewCell('a', twin.StyleDefault.Foreground(twin.NewColor16(0))),
		twin.NewCell('b', twin.StyleDefault.Foreground(twin.NewColor16(1))),
		twin.NewCell('c', twin.StyleDefault.Foreground(twin.NewColor16(2))),
		twin.NewCell('d', twin.StyleDefault.Foreground(twin.NewColor16(3))),
		twin.NewCell('e', twin.StyleDefault.Foreground(twin.NewColor16(4))),
		twin.NewCell('f', twin.StyleDefault.Foreground(twin.NewColor16(5))),
		twin.NewCell('g', twin.StyleDefault.Foreground(twin.NewColor16(6))),
		twin.NewCell('h', twin.StyleDefault.Foreground(twin.NewColor16(7))),
		twin.NewCell('i', twin.StyleDefault),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		logDifference(t, expected, contents[pos])
	}
}

func TestBrokenUtf8(t *testing.T) {
	// The broken UTF8 character in the middle is based on "©" = 0xc2a9
	reader := NewReaderFromStream("", strings.NewReader("abc\xc2def"))

	var answers = []twin.Cell{
		twin.NewCell('a', twin.StyleDefault),
		twin.NewCell('b', twin.StyleDefault),
		twin.NewCell('c', twin.StyleDefault),
		twin.NewCell('?', twin.StyleDefault.Foreground(twin.NewColor16(7)).Background(twin.NewColor16(1))),
		twin.NewCell('d', twin.StyleDefault),
		twin.NewCell('e', twin.StyleDefault),
		twin.NewCell('f', twin.StyleDefault),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		logDifference(t, expected, contents[pos])
	}
}

func startPaging(t *testing.T, reader *Reader) *twin.FakeScreen {
	err := reader._wait()
	if err != nil {
		panic(err)
	}

	screen := twin.NewFakeScreen(20, 10)
	pager := NewPager(reader)
	pager.ShowLineNumbers = false

	// Tell our Pager to quit immediately
	pager.Quit()

	// Except for just quitting, this also associates our FakeScreen with the Pager
	pager.StartPaging(screen)

	// This makes sure at least one frame gets rendered
	pager._Redraw("")

	return screen
}

// assertIndexOfFirstX verifies the (zero-based) index of the first 'x'
func assertIndexOfFirstX(t *testing.T, s string, expectedIndex int) {
	reader := NewReaderFromStream("", strings.NewReader(s))

	contents := startPaging(t, reader).GetRow(0)
	for pos, cell := range contents {
		if cell.Rune != 'x' {
			continue
		}

		if pos == expectedIndex {
			// Success!
			return
		}

		t.Errorf("Expected first 'x' to be at (zero-based) index %d, but was at %d: \"%s\"",
			expectedIndex, pos, strings.ReplaceAll(s, "\x09", "<TAB>"))
		return
	}

	panic("No 'x' found")
}

func TestTabHandling(t *testing.T) {
	assertIndexOfFirstX(t, "x", 0)

	assertIndexOfFirstX(t, "\x09x", 4)
	assertIndexOfFirstX(t, "\x09\x09x", 8)

	assertIndexOfFirstX(t, "J\x09x", 4)
	assertIndexOfFirstX(t, "Jo\x09x", 4)
	assertIndexOfFirstX(t, "Joh\x09x", 4)
	assertIndexOfFirstX(t, "Joha\x09x", 8)
	assertIndexOfFirstX(t, "Johan\x09x", 8)

	assertIndexOfFirstX(t, "\x09J\x09x", 8)
	assertIndexOfFirstX(t, "\x09Jo\x09x", 8)
	assertIndexOfFirstX(t, "\x09Joh\x09x", 8)
	assertIndexOfFirstX(t, "\x09Joha\x09x", 12)
	assertIndexOfFirstX(t, "\x09Johan\x09x", 12)
}

func TestCodeHighlighting(t *testing.T) {
	// From: https://coderwall.com/p/_fmbug/go-get-path-to-current-file
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Getting current filename failed")
	}

	reader, err := NewReaderFromFilename(filename, *styles.Native, formatters.TTY16m)
	if err != nil {
		panic(err)
	}

	packageKeywordStyle := twin.StyleDefault.WithAttr(twin.AttrBold).Foreground(twin.NewColorHex(0x6AB825))
	packageNameStyle := twin.StyleDefault.Foreground(twin.NewColorHex(0xD0D0D0))
	var answers = []twin.Cell{
		twin.NewCell('p', packageKeywordStyle),
		twin.NewCell('a', packageKeywordStyle),
		twin.NewCell('c', packageKeywordStyle),
		twin.NewCell('k', packageKeywordStyle),
		twin.NewCell('a', packageKeywordStyle),
		twin.NewCell('g', packageKeywordStyle),
		twin.NewCell('e', packageKeywordStyle),
		twin.NewCell(' ', packageNameStyle),
		twin.NewCell('m', packageNameStyle),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		logDifference(t, expected, contents[pos])
	}
}

func resetManPageFormat() {
	manPageBold = twin.StyleDefault.WithAttr(twin.AttrBold)
	manPageUnderline = twin.StyleDefault.WithAttr(twin.AttrUnderline)
}

func testManPageFormatting(t *testing.T, input string, expected twin.Cell) {
	reader := NewReaderFromStream("", strings.NewReader(input))

	// Without these three lines the man page tests will fail if either of these
	// environment variables are set when the tests are run.
	if err := os.Setenv("LESS_TERMCAP_md", ""); err != nil {
		panic(err)
	}
	if err := os.Setenv("LESS_TERMCAP_us", ""); err != nil {
		panic(err)
	}
	resetManPageFormat()

	contents := startPaging(t, reader).GetRow(0)
	logDifference(t, expected, contents[0])
	assert.Equal(t, contents[1].Rune, ' ')
}

func TestManPageFormatting(t *testing.T) {
	testManPageFormatting(t, "N\x08N", twin.NewCell('N', twin.StyleDefault.WithAttr(twin.AttrBold)))
	testManPageFormatting(t, "_\x08x", twin.NewCell('x', twin.StyleDefault.WithAttr(twin.AttrUnderline)))

	// Corner cases
	testManPageFormatting(t, "\x08", twin.NewCell('<', twin.StyleDefault.Foreground(twin.NewColor16(7)).Background(twin.NewColor16(1))))

	// FIXME: Test two consecutive backspaces

	// FIXME: Test backspace between two uncombinable characters
}

func TestToPattern(t *testing.T) {
	assert.Assert(t, toPattern("") == nil)

	// Test regexp matching
	assert.Assert(t, toPattern("G.*S").MatchString("GRIIIS"))
	assert.Assert(t, !toPattern("G.*S").MatchString("gRIIIS"))

	// Test case insensitive regexp matching
	assert.Assert(t, toPattern("g.*s").MatchString("GRIIIS"))
	assert.Assert(t, toPattern("g.*s").MatchString("gRIIIS"))

	// Test non-regexp matching
	assert.Assert(t, toPattern(")G").MatchString(")G"))
	assert.Assert(t, !toPattern(")G").MatchString(")g"))

	// Test case insensitive non-regexp matching
	assert.Assert(t, toPattern(")g").MatchString(")G"))
	assert.Assert(t, toPattern(")g").MatchString(")g"))
}

func TestFindFirstLineOneBasedSimple(t *testing.T) {
	reader := NewReaderFromStream("", strings.NewReader("AB"))
	pager := NewPager(reader)

	// Wait for reader to finish reading
	<-reader.done

	pager.searchPattern = toPattern("AB")

	hitLine := pager._FindFirstHitLineOneBased(1, false)
	assert.Check(t, *hitLine == 1)
}

func TestFindFirstLineOneBasedAnsi(t *testing.T) {
	reader := NewReaderFromStream("", strings.NewReader("A\x1b[30mB"))
	pager := NewPager(reader)

	// Wait for reader to finish reading
	<-reader.done

	pager.searchPattern = toPattern("AB")

	hitLine := pager._FindFirstHitLineOneBased(1, false)
	assert.Check(t, *hitLine == 1)
}

// Converts a cell row to a plain string and removes trailing whitespace.
func rowToString(row []twin.Cell) string {
	rowString := ""
	for _, cell := range row {
		rowString += string(cell.Rune)
	}

	return strings.TrimRight(rowString, " ")
}

func TestScrollToBottomWrapNextToLastLine(t *testing.T) {
	reader := NewReaderFromStream("",
		strings.NewReader("first line\nline two will be wrapped\nhere's the last line"))
	pager := NewPager(reader)
	pager.WrapLongLines = true
	pager.ShowLineNumbers = false

	// Wait for reader to finish reading
	<-reader.done

	// This is what we're testing really
	pager._ScrollToEnd()

	// Heigh 3 = two lines of contents + one footer
	screen := twin.NewFakeScreen(10, 3)

	// Exit immediately
	pager.Quit()

	// Get contents onto our fake screen
	pager.StartPaging(screen)
	pager._Redraw("")

	lastVisibleRow := screen.GetRow(1)
	lastVisibleRowString := rowToString(lastVisibleRow)
	assert.Equal(t, lastVisibleRowString, "last line")
}

func TestScrollToBottomWrapLastLine(t *testing.T) {
	reader := NewReaderFromStream("",
		strings.NewReader("this line will be wrapped into four"))
	pager := NewPager(reader)
	pager.WrapLongLines = true
	pager.ShowLineNumbers = false

	// Wait for reader to finish reading
	<-reader.done

	// This is what we're testing really
	pager._ScrollToEnd()

	// Heigh 3 = two lines of contents + one footer
	screen := twin.NewFakeScreen(10, 3)

	// Exit immediately
	pager.Quit()

	// Get contents onto our fake screen
	pager.StartPaging(screen)
	pager._Redraw("")

	lastVisibleRow := screen.GetRow(1)
	lastVisibleRowString := rowToString(lastVisibleRow)
	assert.Equal(t, lastVisibleRowString, "into four")
}

func benchmarkSearch(b *testing.B, highlighted bool) {
	// Pick a go file so we get something with highlighting
	_, sourceFilename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Getting current filename failed")
	}

	// Read one copy of the example input
	var fileContents string
	if highlighted {
		highlightedSourceCode, err := highlight(sourceFilename, true, *styles.Native, formatters.TTY16m)
		if err != nil {
			panic(err)
		}
		if highlightedSourceCode == nil {
			panic("Highlighting didn't want to, returned nil")
		}
		fileContents = *highlightedSourceCode
	} else {
		sourceBytes, err := os.ReadFile(sourceFilename)
		if err != nil {
			panic(err)
		}
		fileContents = string(sourceBytes)
	}

	// Duplicate data N times
	testString := ""
	for n := 0; n < b.N; n++ {
		testString += fileContents
	}

	reader := NewReaderFromText("hello", testString)
	pager := NewPager(reader)

	// The [] around the 't' is there to make sure it doesn't match, remember
	// we're searching through this very file.
	pager.searchPattern = regexp.MustCompile("This won'[t] match anything")

	// Wait for reader to finish reading...
	<-reader.done

	// ... and finish highlighting
	<-reader.highlightingDone

	// I hope forcing a GC here will make numbers more predictable
	runtime.GC()

	b.ResetTimer()

	// This test will search through all the N copies we made of our file
	hitLine := pager._FindFirstHitLineOneBased(1, false)

	if hitLine != nil {
		panic(fmt.Errorf("This test is meant to scan the whole file without finding anything"))
	}
}

// How long does it take to search a highlighted file for some regex?
//
// Run with: go test -run='^$' -bench=. . ./...
func BenchmarkHighlightedSearch(b *testing.B) {
	benchmarkSearch(b, true)
}

// How long does it take to search a plain text file for some regex?
//
// Search performance was a problem for me when I had a 600MB file to search in.
//
// Run with: go test -run='^$' -bench=. . ./...
func BenchmarkPlainTextSearch(b *testing.B) {
	benchmarkSearch(b, false)
}
