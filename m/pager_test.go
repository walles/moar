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

// NOTE: You can find related tests in screenLines_test.go.

const blueBackgroundClearToEol0 = "\x1b[44m\x1b[0K" // With 0 before the K, should clear to EOL
const blueBackgroundClearToEol = "\x1b[44m\x1b[K"   // No 0 before the K, should also clear to EOL

func TestUnicodeRendering(t *testing.T) {
	reader := NewReaderFromText("", "åäö")

	var answers = []twin.StyledRune{
		twin.NewStyledRune('å', twin.StyleDefault),
		twin.NewStyledRune('ä', twin.StyleDefault),
		twin.NewStyledRune('ö', twin.StyleDefault),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		assertRunesEqual(t, expected, contents[pos])
	}
}

func assertRunesEqual(t *testing.T, expected twin.StyledRune, actual twin.StyledRune) {
	if actual.Rune == expected.Rune && actual.Style == expected.Style {
		return
	}

	t.Errorf("Expected %v, got %v", expected, actual)
}

func TestFgColorRendering(t *testing.T) {
	reader := NewReaderFromText("",
		"\x1b[30ma\x1b[31mb\x1b[32mc\x1b[33md\x1b[34me\x1b[35mf\x1b[36mg\x1b[37mh\x1b[0mi")

	var answers = []twin.StyledRune{
		twin.NewStyledRune('a', twin.StyleDefault.WithForeground(twin.NewColor16(0))),
		twin.NewStyledRune('b', twin.StyleDefault.WithForeground(twin.NewColor16(1))),
		twin.NewStyledRune('c', twin.StyleDefault.WithForeground(twin.NewColor16(2))),
		twin.NewStyledRune('d', twin.StyleDefault.WithForeground(twin.NewColor16(3))),
		twin.NewStyledRune('e', twin.StyleDefault.WithForeground(twin.NewColor16(4))),
		twin.NewStyledRune('f', twin.StyleDefault.WithForeground(twin.NewColor16(5))),
		twin.NewStyledRune('g', twin.StyleDefault.WithForeground(twin.NewColor16(6))),
		twin.NewStyledRune('h', twin.StyleDefault.WithForeground(twin.NewColor16(7))),
		twin.NewStyledRune('i', twin.StyleDefault),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		assertRunesEqual(t, expected, contents[pos])
	}
}

func TestPageEmpty(t *testing.T) {
	// "---" is the eofSpinner of pager.go
	assert.Equal(t, "---", renderTextLine(""))
}

func TestBrokenUtf8(t *testing.T) {
	// The broken UTF8 character in the middle is based on "©" = 0xc2a9
	reader := NewReaderFromText("", "abc\xc2def")

	var answers = []twin.StyledRune{
		twin.NewStyledRune('a', twin.StyleDefault),
		twin.NewStyledRune('b', twin.StyleDefault),
		twin.NewStyledRune('c', twin.StyleDefault),
		twin.NewStyledRune('?', twin.StyleDefault.WithForeground(twin.NewColor16(7)).WithBackground(twin.NewColor16(1))),
		twin.NewStyledRune('d', twin.StyleDefault),
		twin.NewStyledRune('e', twin.StyleDefault),
		twin.NewStyledRune('f', twin.StyleDefault),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		assertRunesEqual(t, expected, contents[pos])
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

// Set style to "native" and use the TTY16m formatter
func startPagingWithTerminalFg(t *testing.T, reader *Reader, withTerminalFg bool) *twin.FakeScreen {
	err := reader._wait()
	if err != nil {
		t.Fatalf("Failed waiting for reader: %v", err)
	}

	screen := twin.NewFakeScreen(20, 10)
	pager := NewPager(reader)
	pager.ShowLineNumbers = false
	pager.WithTerminalFg = withTerminalFg

	// Tell our Pager to quit immediately
	pager.Quit()

	// Except for just quitting, this also associates our FakeScreen with the Pager
	pager.StartPaging(screen, styles.Get("native"), &formatters.TTY16m)

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

	reader, err := NewReaderFromFilename(filename, formatters.TTY16m, ReaderOptions{Style: styles.Get("native")})
	assert.NilError(t, err)
	assert.NilError(t, reader._wait())

	packageKeywordStyle := twin.StyleDefault.WithAttr(twin.AttrBold).WithForeground(twin.NewColorHex(0x6AB825))
	packageNameStyle := twin.StyleDefault.WithForeground(twin.NewColorHex(0xD0D0D0))
	var answers = []twin.StyledRune{
		twin.NewStyledRune('p', packageKeywordStyle),
		twin.NewStyledRune('a', packageKeywordStyle),
		twin.NewStyledRune('c', packageKeywordStyle),
		twin.NewStyledRune('k', packageKeywordStyle),
		twin.NewStyledRune('a', packageKeywordStyle),
		twin.NewStyledRune('g', packageKeywordStyle),
		twin.NewStyledRune('e', packageKeywordStyle),
		twin.NewStyledRune(' ', packageNameStyle),
		twin.NewStyledRune('m', packageNameStyle),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		assertRunesEqual(t, expected, contents[pos])
	}
}

func TestCodeHighlight_compressed(t *testing.T) {
	// Same as TestCodeHighlighting but with "compressed-markdown.md.gz"
	reader, err := NewReaderFromFilename("../sample-files/compressed-markdown.md.gz", formatters.TTY16m, ReaderOptions{Style: styles.Get("native")})
	assert.NilError(t, err)
	assert.NilError(t, reader._wait())

	markdownHeading1Style := twin.StyleDefault.WithAttr(twin.AttrBold).WithForeground(twin.NewColorHex(0xffffff))
	var answers = []twin.StyledRune{
		twin.NewStyledRune('#', markdownHeading1Style),
		twin.NewStyledRune(' ', markdownHeading1Style),
		twin.NewStyledRune('M', markdownHeading1Style),
		twin.NewStyledRune('a', markdownHeading1Style),
		twin.NewStyledRune('r', markdownHeading1Style),
		twin.NewStyledRune('k', markdownHeading1Style),
		twin.NewStyledRune('d', markdownHeading1Style),
		twin.NewStyledRune('o', markdownHeading1Style),
		twin.NewStyledRune('w', markdownHeading1Style),
		twin.NewStyledRune('n', markdownHeading1Style),
	}

	contents := startPaging(t, reader).GetRow(0)
	for pos, expected := range answers {
		assertRunesEqual(t, expected, contents[pos])
	}
}

// Regression test for:
// https://github.com/walles/moar/issues/236#issuecomment-2282677792
//
// Sample file sysctl.h from:
// https://github.com/fastfetch-cli/fastfetch/blob/f9597eba39d6afd278eeca2f2972f73a7e54f111/src/common/sysctl.h
func TestCodeHighlightingIncludes(t *testing.T) {
	reader, err := NewReaderFromFilename("../sample-files/sysctl.h", formatters.TTY16m, ReaderOptions{Style: styles.Get("native")})
	assert.NilError(t, err)
	assert.NilError(t, reader._wait())

	screen := startPaging(t, reader)
	firstIncludeLine := screen.GetRow(2)
	secondIncludeLine := screen.GetRow(3)

	// Both should start with "#include" colored the same way
	assertRunesEqual(t, firstIncludeLine[0], secondIncludeLine[0])
}

func TestUnicodePrivateUse(t *testing.T) {
	// This character lives in a Private Use Area:
	// https://codepoints.net/U+f244
	//
	// It's used by Font Awesome as "fa-battery-empty":
	// https://fontawesome.com/v4/icon/battery-empty
	char := '\uf244'

	reader := NewReaderFromText("hello", string(char))
	renderedRune := startPaging(t, reader).GetRow(0)[0]

	// Make sure we display this character unmodified
	assertRunesEqual(t, twin.NewStyledRune(char, twin.StyleDefault), renderedRune)
}

func resetManPageFormat() {
	textstyles.ManPageBold = twin.StyleDefault.WithAttr(twin.AttrBold)
	textstyles.ManPageUnderline = twin.StyleDefault.WithAttr(twin.AttrUnderline)
}

func testManPageFormatting(t *testing.T, input string, expected twin.StyledRune) {
	reader := NewReaderFromText("", input)

	// Without these lines the man page tests will fail if either of these
	// environment variables are set when the tests are run.
	assert.NilError(t, os.Setenv("LESS_TERMCAP_md", ""))
	assert.NilError(t, os.Setenv("LESS_TERMCAP_us", ""))
	assert.NilError(t, os.Setenv("LESS_TERMCAP_so", ""))
	resetManPageFormat()

	contents := startPaging(t, reader).GetRow(0)
	assertRunesEqual(t, expected, contents[0])
	assert.Equal(t, contents[1].Rune, ' ')
}

func TestManPageFormatting(t *testing.T) {
	testManPageFormatting(t, "n\x08n", twin.NewStyledRune('n', twin.StyleDefault.WithAttr(twin.AttrBold)))
	testManPageFormatting(t, "_\x08x", twin.NewStyledRune('x', twin.StyleDefault.WithAttr(twin.AttrUnderline)))

	// Non-breaking space UTF-8 encoded (0xc2a0) should render as a non-breaking unicode space (0xa0)
	testManPageFormatting(t, string([]byte{0xc2, 0xa0}), twin.NewStyledRune(rune(0xa0), twin.StyleDefault))

	// Corner cases
	testManPageFormatting(t, "\x08", twin.NewStyledRune('<', twin.StyleDefault.WithForeground(twin.NewColor16(7)).WithBackground(twin.NewColor16(1))))

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

	assert.NilError(t, pager.reader._wait())

	pager.searchPattern = toPattern("AB")

	hit := pager.findFirstHit(linenumbers.LineNumber{}, nil, false)
	assert.Assert(t, hit.internalDontTouch.lineNumber.IsZero())
	assert.Equal(t, hit.internalDontTouch.deltaScreenLines, 0)
}

func TestFindFirstHitAnsi(t *testing.T) {
	reader := NewReaderFromText("", "A\x1b[30mB")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 10)

	assert.NilError(t, pager.reader._wait())

	pager.searchPattern = toPattern("AB")

	hit := pager.findFirstHit(linenumbers.LineNumber{}, nil, false)
	assert.Assert(t, hit.internalDontTouch.lineNumber.IsZero())
	assert.Equal(t, hit.internalDontTouch.deltaScreenLines, 0)
}

func TestFindFirstHitNoMatch(t *testing.T) {
	reader := NewReaderFromText("TestFindFirstHitSimple", "AB")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 10)

	assert.NilError(t, pager.reader._wait())

	pager.searchPattern = toPattern("this pattern should not be found")

	hit := pager.findFirstHit(linenumbers.LineNumber{}, nil, false)
	assert.Assert(t, hit == nil)
}

func TestFindFirstHitNoMatchBackwards(t *testing.T) {
	reader := NewReaderFromText("TestFindFirstHitSimple", "AB")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 10)

	assert.NilError(t, pager.reader._wait())

	pager.searchPattern = toPattern("this pattern should not be found")
	theEnd := *linenumbers.LineNumberFromLength(reader.GetLineCount())

	hit := pager.findFirstHit(theEnd, nil, true)
	assert.Assert(t, hit == nil)
}

// Converts a cell row to a plain string and removes trailing whitespace.
func rowToString(row []twin.StyledRune) string {
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

	assert.NilError(t, pager.reader._wait())

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
	assertRunesEqual(t, twin.NewStyledRune('X', twin.StyleDefault), lastContentsLine[firstContentsColumn])
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

	pager.mode.onKey(twin.KeyUp)
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
	for _, fileName := range getTestFiles(t) {
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

			myReader := NewReaderFromStream(fileName, file, nil, ReaderOptions{Style: &chroma.Style{}})
			assert.NilError(t, myReader._wait())

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
	var expected []twin.StyledRune
	for len(expected) < screenWidth {
		expected = append(expected,
			twin.NewStyledRune(' ', twin.StyleDefault.WithBackground(twin.NewColor16(4))),
		)
	}

	actual := screen.GetRow(0)
	assert.DeepEqual(t, actual, expected, cmp.AllowUnexported(twin.Style{}))
}

// Validate rendering of https://en.wikipedia.org/wiki/ANSI_escape_code#EL
func TestClearToEndOfLine_ClearFromNotStart(t *testing.T) {
	screen := startPaging(t, NewReaderFromText("TestClearToEol", "a"+blueBackgroundClearToEol))

	screenWidth, _ := screen.Size()
	expected := []twin.StyledRune{
		twin.NewStyledRune('a', twin.StyleDefault),
	}
	for len(expected) < screenWidth {
		expected = append(expected,
			twin.NewStyledRune(' ', twin.StyleDefault.WithBackground(twin.NewColor16(4))),
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
	var expected []twin.StyledRune
	for len(expected) < screenWidth {
		expected = append(expected,
			twin.NewStyledRune(' ', twin.StyleDefault.WithBackground(twin.NewColor16(4))),
		)
	}

	actual := screen.GetRow(0)
	assert.DeepEqual(t, actual, expected, cmp.AllowUnexported(twin.Style{}))
}

// Render a line of text on our 20 cell wide screen
func renderTextLine(text string) string {
	reader := NewReaderFromText("renderTextLine", text)
	screen := startPaging(nil, reader)
	return rowToString(screen.GetRow(0))
}

// Ref: https://github.com/walles/moar/issues/243
func TestPageWideChars(t *testing.T) {
	// Both of these characters are 2 cells wide on a terminal
	const monospaced4cells = "上午"
	const monospaced8cells = monospaced4cells + monospaced4cells
	const monospaced16cells = monospaced8cells + monospaced8cells
	const monospaced20cells = monospaced16cells + monospaced4cells
	const monospaced24cells = monospaced16cells + monospaced8cells

	// Cut the line in the middle of a wide character
	const monospaced18cells = monospaced16cells + "上"
	assert.Equal(t, monospaced18cells+" >", renderTextLine(monospaced24cells))

	// Just the right length, no cutting
	assert.Equal(t, monospaced20cells, renderTextLine(monospaced20cells))

	// Cut this line after a whide character
	assert.Equal(t, "x"+monospaced18cells+">", renderTextLine("x"+monospaced24cells))
}

func TestTerminalFg(t *testing.T) {
	reader := NewReaderFromText("", "x")

	var styleAnswer = twin.NewStyledRune('x', twin.StyleDefault.WithForeground(twin.NewColor24Bit(0xd0, 0xd0, 0xd0)))
	var terminalAnswer = twin.NewStyledRune('x', twin.StyleDefault)

	assertRunesEqual(t, styleAnswer, startPagingWithTerminalFg(t, reader, false).GetRow(0)[0])
	assertRunesEqual(t, terminalAnswer, startPagingWithTerminalFg(t, reader, true).GetRow(0)[0])
}

func benchmarkSearch(b *testing.B, highlighted bool) {
	// Pick a go file so we get something with highlighting
	_, sourceFilename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Getting current filename failed")
	}

	sourceBytes, err := os.ReadFile(sourceFilename)
	assert.NilError(b, err)
	fileContents := string(sourceBytes)

	// Read one copy of the example input
	if highlighted {
		highlightedSourceCode, err := highlight(fileContents, *styles.Get("native"), formatters.TTY16m, lexers.Get("go"))
		assert.NilError(b, err)
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
	hit := pager.findFirstHit(linenumbers.LineNumber{}, nil, false)

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
