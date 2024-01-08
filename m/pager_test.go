package m

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/google/go-cmp/cmp"
	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

//revive:disable:empty-block

const blueBackgroundClearToEol0 = "\x1b[44m\x1b[0K" // With 0 before the K, should clear to EOL
const blueBackgroundClearToEol = "\x1b[44m\x1b[K"   // No 0 before the K, should also clear to EOL

func TestUnicodeRendering(t *testing.T) {
	reader := NewReaderFromText("", "åäö")

	var answers = []twin.Cell{
		twin.NewCell('å', twin.StyleDefault),
		twin.NewCell('ä', twin.StyleDefault),
		twin.NewCell('ö', twin.StyleDefault),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		assertCellsEqual(t, expected, contents[pos])
	}
}

func assertCellsEqual(t *testing.T, expected twin.Cell, actual twin.Cell) {
	if actual.Rune == expected.Rune && actual.Style == expected.Style {
		return
	}

	t.Errorf("Expected %v, got %v", expected, actual)
}

func TestFgColorRendering(t *testing.T) {
	reader := NewReaderFromText("",
		"\x1b[30ma\x1b[31mb\x1b[32mc\x1b[33md\x1b[34me\x1b[35mf\x1b[36mg\x1b[37mh\x1b[0mi")

	var answers = []twin.Cell{
		twin.NewCell('a', twin.StyleDefault.WithForeground(twin.NewColor16(0))),
		twin.NewCell('b', twin.StyleDefault.WithForeground(twin.NewColor16(1))),
		twin.NewCell('c', twin.StyleDefault.WithForeground(twin.NewColor16(2))),
		twin.NewCell('d', twin.StyleDefault.WithForeground(twin.NewColor16(3))),
		twin.NewCell('e', twin.StyleDefault.WithForeground(twin.NewColor16(4))),
		twin.NewCell('f', twin.StyleDefault.WithForeground(twin.NewColor16(5))),
		twin.NewCell('g', twin.StyleDefault.WithForeground(twin.NewColor16(6))),
		twin.NewCell('h', twin.StyleDefault.WithForeground(twin.NewColor16(7))),
		twin.NewCell('i', twin.StyleDefault),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		assertCellsEqual(t, expected, contents[pos])
	}
}

func TestPageEmpty(t *testing.T) {
	reader := NewReaderFromText("", "")

	firstRowCells := startPaging(t, reader).GetRow(0)

	// "---" is the eofSpinner of pager.go
	assert.Equal(t, "---", rowToString(firstRowCells))
}

func TestBrokenUtf8(t *testing.T) {
	// The broken UTF8 character in the middle is based on "©" = 0xc2a9
	reader := NewReaderFromText("", "abc\xc2def")

	var answers = []twin.Cell{
		twin.NewCell('a', twin.StyleDefault),
		twin.NewCell('b', twin.StyleDefault),
		twin.NewCell('c', twin.StyleDefault),
		twin.NewCell('?', twin.StyleDefault.WithForeground(twin.NewColor16(7)).WithBackground(twin.NewColor16(1))),
		twin.NewCell('d', twin.StyleDefault),
		twin.NewCell('e', twin.StyleDefault),
		twin.NewCell('f', twin.StyleDefault),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		assertCellsEqual(t, expected, contents[pos])
	}
}

func startPaging(t *testing.T, reader *Reader) *twin.FakeScreen {
	err := reader._wait()
	if err != nil {
		t.Fatalf("Failed waiting for reader: %v", err)
	}

	screen := twin.NewFakeScreen(20, 10)
	pager := NewPager(reader)
	pager.ShowLineNumbers = false

	// Tell our Pager to quit immediately
	pager.Quit()

	// Except for just quitting, this also associates our FakeScreen with the Pager
	pager.StartPaging(screen, nil, nil)

	// This makes sure at least one frame gets rendered
	pager.redraw("")

	return screen
}

// assertIndexOfFirstX verifies the (zero-based) index of the first 'x'
func assertIndexOfFirstX(t *testing.T, s string, expectedIndex int) {
	reader := NewReaderFromText("", s)

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

	reader, err := NewReaderFromFilename(filename, *styles.Get("native"), formatters.TTY16m, nil)
	if err != nil {
		panic(err)
	}

	packageKeywordStyle := twin.StyleDefault.WithAttr(twin.AttrBold).WithForeground(twin.NewColorHex(0x6AB825))
	packageNameStyle := twin.StyleDefault.WithForeground(twin.NewColorHex(0xD0D0D0))
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
		assertCellsEqual(t, expected, contents[pos])
	}
}

func TestUnicodePrivateUse(t *testing.T) {
	// This character lives in a Private Use Area:
	// https://codepoints.net/U+f244
	//
	// It's used by Font Awesome as "fa-battery-empty":
	// https://fontawesome.com/v4/icon/battery-empty
	char := '\uf244'

	reader := NewReaderFromText("hello", string(char))
	renderedCell := startPaging(t, reader).GetRow(0)[0]

	// Make sure we display this character unmodified
	assertCellsEqual(t, twin.NewCell(char, twin.StyleDefault), renderedCell)
}

func resetManPageFormat() {
	textstyles.ManPageBold = twin.StyleDefault.WithAttr(twin.AttrBold)
	textstyles.ManPageUnderline = twin.StyleDefault.WithAttr(twin.AttrUnderline)
}

func testManPageFormatting(t *testing.T, input string, expected twin.Cell) {
	reader := NewReaderFromText("", input)

	// Without these lines the man page tests will fail if either of these
	// environment variables are set when the tests are run.
	if err := os.Setenv("LESS_TERMCAP_md", ""); err != nil {
		panic(err)
	}
	if err := os.Setenv("LESS_TERMCAP_us", ""); err != nil {
		panic(err)
	}
	if err := os.Setenv("LESS_TERMCAP_so", ""); err != nil {
		panic(err)
	}
	resetManPageFormat()

	contents := startPaging(t, reader).GetRow(0)
	assertCellsEqual(t, expected, contents[0])
	assert.Equal(t, contents[1].Rune, ' ')
}

func TestManPageFormatting(t *testing.T) {
	testManPageFormatting(t, "N\x08N", twin.NewCell('N', twin.StyleDefault.WithAttr(twin.AttrBold)))
	testManPageFormatting(t, "_\x08x", twin.NewCell('x', twin.StyleDefault.WithAttr(twin.AttrUnderline)))

	// Corner cases
	testManPageFormatting(t, "\x08", twin.NewCell('<', twin.StyleDefault.WithForeground(twin.NewColor16(7)).WithBackground(twin.NewColor16(1))))

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

func TestFindFirstHitSimple(t *testing.T) {
	reader := NewReaderFromText("TestFindFirstHitSimple", "AB")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 10)

	// Wait for reader to finish reading
	for !reader.done.Load() {
	}

	pager.searchPattern = toPattern("AB")

	hit := pager.findFirstHit(newScrollPosition("TestFindFirstHitSimple"), false)
	assert.Assert(t, hit.internalDontTouch.lineNumber.IsZero())
	assert.Equal(t, hit.internalDontTouch.deltaScreenLines, 0)
}

func TestFindFirstHitAnsi(t *testing.T) {
	reader := NewReaderFromText("", "A\x1b[30mB")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 10)

	// Wait for reader to finish reading
	for !reader.done.Load() {
	}

	pager.searchPattern = toPattern("AB")

	hit := pager.findFirstHit(newScrollPosition("TestFindFirstHitSimple"), false)
	assert.Assert(t, hit.internalDontTouch.lineNumber.IsZero())
	assert.Equal(t, hit.internalDontTouch.deltaScreenLines, 0)
}

func TestFindFirstHitNoMatch(t *testing.T) {
	reader := NewReaderFromText("TestFindFirstHitSimple", "AB")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 10)

	// Wait for reader to finish reading
	for !reader.done.Load() {
	}

	pager.searchPattern = toPattern("this pattern should not be found")

	hit := pager.findFirstHit(newScrollPosition("TestFindFirstHitSimple"), false)
	assert.Assert(t, hit == nil)
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
	reader := NewReaderFromText("",
		"first line\nline two will be wrapped\nhere's the last line")

	// Heigh 3 = two lines of contents + one footer
	screen := twin.NewFakeScreen(10, 3)

	pager := NewPager(reader)
	pager.WrapLongLines = true
	pager.ShowLineNumbers = false
	pager.screen = screen

	// Wait for reader to finish reading
	for !reader.done.Load() {
	}

	// This is what we're testing really
	pager.scrollToEnd()

	// Exit immediately
	pager.Quit()

	// Get contents onto our fake screen
	pager.StartPaging(screen, nil, nil)
	pager.redraw("")

	actual := strings.Join([]string{
		rowToString(screen.GetRow(0)),
		rowToString(screen.GetRow(1)),
		rowToString(screen.GetRow(2)),
	}, "\n")
	expected := strings.Join([]string{
		"here's the",
		"last line",
		"3 lines  1", // "3 lines 100%" clipped after 10 characters (screen width)
	}, "\n")
	assert.Equal(t, actual, expected)
}

// Repro for https://github.com/walles/moar/issues/105
func TestScrollToEndLongInput(t *testing.T) {
	const lineCount = 10100 // At least five digits

	// "X" marks the spot
	reader := NewReaderFromText("test", strings.Repeat(".\n", lineCount-1)+"X")
	pager := NewPager(reader)
	pager.ShowLineNumbers = true

	// Tell our Pager to quit immediately
	pager.Quit()

	// Connect the pager with a screen
	const screenHeight = 10
	screen := twin.NewFakeScreen(20, screenHeight)
	pager.StartPaging(screen, nil, nil)

	// This is what we're really testing
	pager.scrollToEnd()

	// This makes sure at least one frame gets rendered
	pager.redraw("")

	// The last screen line holds the status field, and the next to last screen
	// line holds the last contents line.
	lastContentsLine := screen.GetRow(screenHeight - 2)
	firstContentsColumn := len("10_100 ")
	assertCellsEqual(t, twin.NewCell('X', twin.StyleDefault), lastContentsLine[firstContentsColumn])
}

func TestIsScrolledToEnd_LongFile(t *testing.T) {
	// Six lines of contents
	reader := NewReaderFromText("Testing", "a\nb\nc\nd\ne\nf\n")

	// Three lines screen
	screen := twin.NewFakeScreen(20, 3)

	// Create the pager
	pager := NewPager(reader)
	pager.screen = screen

	assert.Equal(t, false, pager.isScrolledToEnd())

	pager.scrollToEnd()
	assert.Equal(t, true, pager.isScrolledToEnd())
}

func TestIsScrolledToEnd_ShortFile(t *testing.T) {
	// Three lines of contents
	reader := NewReaderFromText("Testing", "a\nb\nc")

	// Six lines screen
	screen := twin.NewFakeScreen(20, 6)

	// Create the pager
	pager := NewPager(reader)
	pager.screen = screen

	assert.Equal(t, true, pager.isScrolledToEnd())

	pager.scrollToEnd()
	assert.Equal(t, true, pager.isScrolledToEnd())
}

func TestIsScrolledToEnd_ExactFile(t *testing.T) {
	// Three lines of contents
	reader := NewReaderFromText("Testing", "a\nb\nc")

	// Three lines screen
	screen := twin.NewFakeScreen(20, 3)

	// Create the pager
	pager := NewPager(reader)
	pager.screen = screen
	pager.ShowStatusBar = false

	assert.Equal(t, true, pager.isScrolledToEnd())

	pager.scrollToEnd()
	assert.Equal(t, true, pager.isScrolledToEnd())
}

func TestIsScrolledToEnd_WrappedLastLine(t *testing.T) {
	// Three lines of contents
	reader := NewReaderFromText("Testing", "a\nb\nc d e f g h i j k l m n")

	// Three lines screen
	screen := twin.NewFakeScreen(5, 3)

	// Create the pager
	pager := NewPager(reader)
	pager.screen = screen
	pager.WrapLongLines = true

	assert.Equal(t, false, pager.isScrolledToEnd())

	pager.scrollToEnd()
	assert.Equal(t, true, pager.isScrolledToEnd())

	pager.onKey(twin.KeyUp)
	pager.redraw("XXX")
	assert.Equal(t, false, pager.isScrolledToEnd())
}

func TestIsScrolledToEnd_EmptyFile(t *testing.T) {
	// No contents
	reader := NewReaderFromText("Testing", "")

	// Three lines screen
	screen := twin.NewFakeScreen(20, 3)

	// Create the pager
	pager := NewPager(reader)
	pager.screen = screen

	assert.Equal(t, true, pager.isScrolledToEnd())

	pager.scrollToEnd()
	assert.Equal(t, true, pager.isScrolledToEnd())
}

// Verify that we can page all files in ../sample-files/* without crashing
func TestPageSamples(t *testing.T) {
	for _, fileName := range getTestFiles() {
		t.Run(fileName, func(t *testing.T) {
			file, err := os.Open(fileName)
			if err != nil {
				t.Errorf("Error opening file <%s>: %s", fileName, err.Error())
				return
			}
			defer func() {
				if err := file.Close(); err != nil {
					panic(err)
				}
			}()

			myReader := NewReaderFromStream(fileName, file, chroma.Style{}, nil, nil)
			for !myReader.done.Load() {
			}

			pager := NewPager(myReader)
			pager.WrapLongLines = false
			pager.ShowLineNumbers = false

			// Heigh 3 = two lines of contents + one footer
			screen := twin.NewFakeScreen(10, 3)

			// Exit immediately
			pager.Quit()

			// Get contents onto our fake screen
			pager.StartPaging(screen, nil, nil)
			pager.redraw("")

			firstReaderLine := myReader.GetLine(linenumbers.LineNumber{})
			if firstReaderLine == nil {
				return
			}
			firstPagerLine := rowToString(screen.GetRow(0))

			// Handle the case when first line is chopped off to the right
			firstPagerLine = strings.TrimSuffix(firstPagerLine, ">")

			assert.Assert(t,
				strings.HasPrefix(firstReaderLine.Plain(nil), firstPagerLine),
				"\nreader line = <%s>\npager line  = <%s>",
				firstReaderLine.Plain(nil), firstPagerLine,
			)
		})
	}
}

// Validate rendering of https://en.wikipedia.org/wiki/ANSI_escape_code#EL
func TestClearToEndOfLine_ClearFromStart(t *testing.T) {
	screen := startPaging(t, NewReaderFromText("TestClearToEol", blueBackgroundClearToEol))

	screenWidth, _ := screen.Size()
	var expected []twin.Cell
	for len(expected) < screenWidth {
		expected = append(expected,
			twin.NewCell(' ', twin.StyleDefault.WithBackground(twin.NewColor16(4))),
		)
	}

	actual := screen.GetRow(0)
	assert.DeepEqual(t, actual, expected, cmp.AllowUnexported(twin.Style{}))
}

// Validate rendering of https://en.wikipedia.org/wiki/ANSI_escape_code#EL
func TestClearToEndOfLine_ClearFromNotStart(t *testing.T) {
	screen := startPaging(t, NewReaderFromText("TestClearToEol", "a"+blueBackgroundClearToEol))

	screenWidth, _ := screen.Size()
	expected := []twin.Cell{
		twin.NewCell('a', twin.StyleDefault),
	}
	for len(expected) < screenWidth {
		expected = append(expected,
			twin.NewCell(' ', twin.StyleDefault.WithBackground(twin.NewColor16(4))),
		)
	}

	actual := screen.GetRow(0)
	assert.DeepEqual(t, actual, expected, cmp.AllowUnexported(twin.Style{}))
}

// Validate rendering of https://en.wikipedia.org/wiki/ANSI_escape_code#EL
func TestClearToEndOfLine_ClearFromStartScrolledRight(t *testing.T) {
	pager := NewPager(NewReaderFromText("TestClearToEol", blueBackgroundClearToEol0))
	pager.ShowLineNumbers = false

	// Tell our Pager to quit immediately
	pager.Quit()

	// Except for just quitting, this also associates a FakeScreen with the Pager
	screen := twin.NewFakeScreen(3, 10)
	pager.StartPaging(screen, nil, nil)

	// Scroll right, this is what we're testing
	pager.leftColumnZeroBased = 44

	// This makes sure at least one frame gets rendered
	pager.redraw("")

	screenWidth, _ := screen.Size()
	var expected []twin.Cell
	for len(expected) < screenWidth {
		expected = append(expected,
			twin.NewCell(' ', twin.StyleDefault.WithBackground(twin.NewColor16(4))),
		)
	}

	actual := screen.GetRow(0)
	assert.DeepEqual(t, actual, expected, cmp.AllowUnexported(twin.Style{}))
}

func TestGetLineColorPrefix(t *testing.T) {
	assert.Equal(t,
		getLineColorPrefix(styles.Registry["gruvbox"], &formatters.TTY16m),
		"\x1b[38;2;235;219;178m",
	)
}

func TestInitStyle256(t *testing.T) {
	assert.Equal(t,
		getLineColorPrefix(
			styles.Registry["catppuccin-macchiato"],
			&formatters.TTY256), "\x1b[38;5;189m",
	)
}

func benchmarkSearch(b *testing.B, highlighted bool) {
	// Pick a go file so we get something with highlighting
	_, sourceFilename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Getting current filename failed")
	}

	sourceBytes, err := os.ReadFile(sourceFilename)
	if err != nil {
		panic(err)
	}
	fileContents := string(sourceBytes)

	// Read one copy of the example input
	if highlighted {
		highlightedSourceCode, err := highlight(fileContents, *styles.Get("native"), formatters.TTY16m, lexers.Get("go"))
		if err != nil {
			panic(err)
		}
		if highlightedSourceCode == nil {
			panic("Highlighting didn't want to, returned nil")
		}
		fileContents = *highlightedSourceCode
	}

	// Duplicate data N times
	testString := ""
	for n := 0; n < b.N; n++ {
		testString += fileContents
	}

	reader := NewReaderFromText("hello", testString)
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 10)

	// The [] around the 't' is there to make sure it doesn't match, remember
	// we're searching through this very file.
	pager.searchPattern = regexp.MustCompile("This won'[t] match anything")

	// I hope forcing a GC here will make numbers more predictable
	runtime.GC()

	b.ResetTimer()

	// This test will search through all the N copies we made of our file
	hit := pager.findFirstHit(newScrollPosition("benchmarkSearch"), false)

	if hit != nil {
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
