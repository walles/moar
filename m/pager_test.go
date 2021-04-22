package m

import (
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

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
	err := reader._Wait()
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

	reader, err := NewReaderFromFilename(filename)
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
	resetManPageFormatForTesting()

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

func assertTokenRangesEqual(t *testing.T, actual []twin.Cell, expected []twin.Cell) {
	if len(actual) != len(expected) {
		t.Errorf("String lengths mismatch; expected %d but got %d",
			len(expected), len(actual))
	}

	for pos, expectedToken := range expected {
		if pos >= len(expected) || pos >= len(actual) {
			break
		}

		actualToken := actual[pos]
		if actualToken.Rune == expectedToken.Rune && actualToken.Style == expectedToken.Style {
			// Actual == Expected, keep checking
			continue
		}

		t.Errorf("At (0-based) position %d: Expected %v, got %v", pos, expectedToken, actualToken)
	}
}

func TestCreateScreenLineBase(t *testing.T) {
	line := NewLine("")
	screenLine := createScreenLine(0, 3, line, nil)
	assert.Assert(t, len(screenLine) == 0)
}

func TestCreateScreenLineOverflowRight(t *testing.T) {
	line := NewLine("012345")
	screenLine := createScreenLine(0, 3, line, nil)
	assertTokenRangesEqual(t, screenLine, []twin.Cell{
		twin.NewCell('0', twin.StyleDefault),
		twin.NewCell('1', twin.StyleDefault),
		twin.NewCell('>', twin.StyleDefault.WithAttr(twin.AttrReverse)),
	})
}

func TestCreateScreenLineUnderflowLeft(t *testing.T) {
	line := NewLine("012")
	screenLine := createScreenLine(1, 3, line, nil)
	assertTokenRangesEqual(t, screenLine, []twin.Cell{
		twin.NewCell('<', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		twin.NewCell('1', twin.StyleDefault),
		twin.NewCell('2', twin.StyleDefault),
	})
}

func TestCreateScreenLineSearchHit(t *testing.T) {
	pattern, err := regexp.Compile("b")
	if err != nil {
		panic(err)
	}

	line := NewLine("abc")
	screenLine := createScreenLine(0, 3, line, pattern)
	assertTokenRangesEqual(t, screenLine, []twin.Cell{
		twin.NewCell('a', twin.StyleDefault),
		twin.NewCell('b', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		twin.NewCell('c', twin.StyleDefault),
	})
}

func TestCreateScreenLineUtf8SearchHit(t *testing.T) {
	pattern, err := regexp.Compile("ä")
	if err != nil {
		panic(err)
	}

	line := NewLine("åäö")
	screenLine := createScreenLine(0, 3, line, pattern)
	assertTokenRangesEqual(t, screenLine, []twin.Cell{
		twin.NewCell('å', twin.StyleDefault),
		twin.NewCell('ä', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		twin.NewCell('ö', twin.StyleDefault),
	})
}

func TestCreateScreenLineScrolledUtf8SearchHit(t *testing.T) {
	pattern := regexp.MustCompile("ä")

	line := NewLine("ååäö")
	screenLine := createScreenLine(1, 4, line, pattern)

	assertTokenRangesEqual(t, screenLine, []twin.Cell{
		twin.NewCell('<', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		twin.NewCell('å', twin.StyleDefault),
		twin.NewCell('ä', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		twin.NewCell('ö', twin.StyleDefault),
	})
}

func TestCreateScreenLineScrolled2Utf8SearchHit(t *testing.T) {
	pattern := regexp.MustCompile("ä")

	line := NewLine("åååäö")
	screenLine := createScreenLine(2, 4, line, pattern)

	assertTokenRangesEqual(t, screenLine, []twin.Cell{
		twin.NewCell('<', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		twin.NewCell('å', twin.StyleDefault),
		twin.NewCell('ä', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		twin.NewCell('ö', twin.StyleDefault),
	})
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
