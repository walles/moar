package m

import (
	"log"
	"os"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/gdamore/tcell"
	"gotest.tools/assert"
)

func TestUnicodeRendering(t *testing.T) {
	reader := NewReaderFromStream(strings.NewReader("åäö"), nil)
	if err := reader._Wait(); err != nil {
		panic(err)
	}

	var answers = []Token{
		_CreateExpectedCell('å', tcell.StyleDefault),
		_CreateExpectedCell('ä', tcell.StyleDefault),
		_CreateExpectedCell('ö', tcell.StyleDefault),
	}

	contents := _StartPaging(t, reader)
	for pos, expected := range answers {
		expected.LogDifference(t, contents[pos])
	}
}

func (expected Token) LogDifference(t *testing.T, actual tcell.SimCell) {
	if actual.Runes[0] == expected.Rune && actual.Style == expected.Style {
		return
	}

	t.Errorf("Expected '%s'/0x%x, got '%s'/0x%x",
		string(expected.Rune), expected.Style,
		string(actual.Runes[0]), actual.Style)
}

func _CreateExpectedCell(Rune rune, Style tcell.Style) Token {
	return Token{
		Rune:  Rune,
		Style: Style,
	}
}

func TestFgColorRendering(t *testing.T) {
	reader := NewReaderFromStream(strings.NewReader(
		"\x1b[30ma\x1b[31mb\x1b[32mc\x1b[33md\x1b[34me\x1b[35mf\x1b[36mg\x1b[37mh\x1b[0mi"), nil)
	if err := reader._Wait(); err != nil {
		panic(err)
	}

	var answers = []Token{
		_CreateExpectedCell('a', tcell.StyleDefault.Foreground(0)),
		_CreateExpectedCell('b', tcell.StyleDefault.Foreground(1)),
		_CreateExpectedCell('c', tcell.StyleDefault.Foreground(2)),
		_CreateExpectedCell('d', tcell.StyleDefault.Foreground(3)),
		_CreateExpectedCell('e', tcell.StyleDefault.Foreground(4)),
		_CreateExpectedCell('f', tcell.StyleDefault.Foreground(5)),
		_CreateExpectedCell('g', tcell.StyleDefault.Foreground(6)),
		_CreateExpectedCell('h', tcell.StyleDefault.Foreground(7)),
		_CreateExpectedCell('i', tcell.StyleDefault),
	}

	contents := _StartPaging(t, reader)
	for pos, expected := range answers {
		expected.LogDifference(t, contents[pos])
	}
}

func TestBrokenUtf8(t *testing.T) {
	// The broken UTF8 character in the middle is based on "©" = 0xc2a9
	reader := NewReaderFromStream(strings.NewReader(
		"abc\xc2def"), nil)
	if err := reader._Wait(); err != nil {
		panic(err)
	}

	var answers = []Token{
		_CreateExpectedCell('a', tcell.StyleDefault),
		_CreateExpectedCell('b', tcell.StyleDefault),
		_CreateExpectedCell('c', tcell.StyleDefault),
		_CreateExpectedCell('?', tcell.StyleDefault.Foreground(1).Background(7)),
		_CreateExpectedCell('d', tcell.StyleDefault),
		_CreateExpectedCell('e', tcell.StyleDefault),
		_CreateExpectedCell('f', tcell.StyleDefault),
	}

	contents := _StartPaging(t, reader)
	for pos, expected := range answers {
		expected.LogDifference(t, contents[pos])
	}
}

func _StartPaging(t *testing.T, reader *Reader) []tcell.SimCell {
	screen := tcell.NewSimulationScreen("UTF-8")
	pager := NewPager(reader)
	pager.Quit()

	var loglines strings.Builder
	logger := log.New(&loglines, "", 0)
	pager.StartPaging(logger, screen)
	contents, _, _ := screen.GetContents()

	if len(loglines.String()) > 0 {
		t.Logf("%s", loglines.String())
	}

	return contents
}

// _AssertIndexOfFirstX verifies the (zero-based) index of the first 'x'
func _AssertIndexOfFirstX(t *testing.T, s string, expectedIndex int) {
	reader := NewReaderFromStream(strings.NewReader(s), nil)
	if err := reader._Wait(); err != nil {
		panic(err)
	}

	contents := _StartPaging(t, reader)
	for pos, cell := range contents {
		if cell.Runes[0] != 'x' {
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
	_AssertIndexOfFirstX(t, "x", 0)

	_AssertIndexOfFirstX(t, "\x09x", 4)
	_AssertIndexOfFirstX(t, "\x09\x09x", 8)

	_AssertIndexOfFirstX(t, "J\x09x", 4)
	_AssertIndexOfFirstX(t, "Jo\x09x", 4)
	_AssertIndexOfFirstX(t, "Joh\x09x", 4)
	_AssertIndexOfFirstX(t, "Joha\x09x", 8)
	_AssertIndexOfFirstX(t, "Johan\x09x", 8)

	_AssertIndexOfFirstX(t, "\x09J\x09x", 8)
	_AssertIndexOfFirstX(t, "\x09Jo\x09x", 8)
	_AssertIndexOfFirstX(t, "\x09Joh\x09x", 8)
	_AssertIndexOfFirstX(t, "\x09Joha\x09x", 12)
	_AssertIndexOfFirstX(t, "\x09Johan\x09x", 12)
}

// This test assumes highlight is installed:
// http://www.andre-simon.de/zip/download.php
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
	if err := reader._Wait(); err != nil {
		panic(err)
	}

	var answers = []Token{
		_CreateExpectedCell('p', tcell.StyleDefault.Foreground(3)),
		_CreateExpectedCell('a', tcell.StyleDefault.Foreground(3)),
		_CreateExpectedCell('c', tcell.StyleDefault.Foreground(3)),
		_CreateExpectedCell('k', tcell.StyleDefault.Foreground(3)),
		_CreateExpectedCell('a', tcell.StyleDefault.Foreground(3)),
		_CreateExpectedCell('g', tcell.StyleDefault.Foreground(3)),
		_CreateExpectedCell('e', tcell.StyleDefault.Foreground(3)),
		_CreateExpectedCell(' ', tcell.StyleDefault),
		_CreateExpectedCell('m', tcell.StyleDefault),
	}

	contents := _StartPaging(t, reader)
	for pos, expected := range answers {
		expected.LogDifference(t, contents[pos])
	}
}

func _TestManPageFormatting(t *testing.T, input string, expected Token) {
	reader := NewReaderFromStream(strings.NewReader(input), nil)
	if err := reader._Wait(); err != nil {
		panic(err)
	}

	// Without these three lines the man page tests will fail if either of these
	// environment variables are set when the tests are run.
	os.Setenv("LESS_TERMCAP_md", "")
	os.Setenv("LESS_TERMCAP_us", "")
	_ResetManPageFormatForTesting()

	contents := _StartPaging(t, reader)
	expected.LogDifference(t, contents[0])
	assert.Equal(t, contents[1].Runes[0], ' ')
}

func TestManPageFormatting(t *testing.T) {
	_TestManPageFormatting(t, "N\x08N", _CreateExpectedCell('N', tcell.StyleDefault.Bold(true)))
	_TestManPageFormatting(t, "_\x08x", _CreateExpectedCell('x', tcell.StyleDefault.Underline(true)))

	// Corner cases
	_TestManPageFormatting(t, "\x08", _CreateExpectedCell('<', tcell.StyleDefault.Foreground(1).Background(7)))

	// FIXME: Test two consecutive backspaces

	// FIXME: Test backspace between two uncombinable characters
}

func TestToPattern(t *testing.T) {
	assert.Assert(t, ToPattern("") == nil)

	// Test regexp matching
	assert.Assert(t, ToPattern("G.*S").MatchString("GRIIIS"))
	assert.Assert(t, !ToPattern("G.*S").MatchString("gRIIIS"))

	// Test case insensitive regexp matching
	assert.Assert(t, ToPattern("g.*s").MatchString("GRIIIS"))
	assert.Assert(t, ToPattern("g.*s").MatchString("gRIIIS"))

	// Test non-regexp matching
	assert.Assert(t, ToPattern(")G").MatchString(")G"))
	assert.Assert(t, !ToPattern(")G").MatchString(")g"))

	// Test case insensitive non-regexp matching
	assert.Assert(t, ToPattern(")g").MatchString(")G"))
	assert.Assert(t, ToPattern(")g").MatchString(")g"))
}

func assertTokenRangesEqual(t *testing.T, actual []Token, expected []Token) {
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

		t.Errorf("At (0-based) position %d: Expected '%s'/0x%x, got '%s'/0x%x",
			pos,
			string(expectedToken.Rune), expectedToken.Style,
			string(actualToken.Rune), actualToken.Style)
	}
}

func TestCreateScreenLineBase(t *testing.T) {
	line := _CreateScreenLine(nil, 0, 0, 3, "", nil)
	assert.Assert(t, len(line) == 0)
}

func TestCreateScreenLineOverflowRight(t *testing.T) {
	line := _CreateScreenLine(nil, 0, 0, 3, "012345", nil)
	assertTokenRangesEqual(t, line, []Token{
		_CreateExpectedCell('0', tcell.StyleDefault),
		_CreateExpectedCell('1', tcell.StyleDefault),
		_CreateExpectedCell('>', tcell.StyleDefault.Reverse(true)),
	})
}

func TestCreateScreenLineUnderflowLeft(t *testing.T) {
	line := _CreateScreenLine(nil, 0, 1, 3, "012", nil)
	assertTokenRangesEqual(t, line, []Token{
		_CreateExpectedCell('<', tcell.StyleDefault.Reverse(true)),
		_CreateExpectedCell('1', tcell.StyleDefault),
		_CreateExpectedCell('2', tcell.StyleDefault),
	})
}

func TestCreateScreenLineSearchHit(t *testing.T) {
	pattern, err := regexp.Compile("b")
	if err != nil {
		panic(err)
	}

	line := _CreateScreenLine(nil, 0, 0, 3, "abc", pattern)
	assertTokenRangesEqual(t, line, []Token{
		_CreateExpectedCell('a', tcell.StyleDefault),
		_CreateExpectedCell('b', tcell.StyleDefault.Reverse(true)),
		_CreateExpectedCell('c', tcell.StyleDefault),
	})
}

func TestCreateScreenLineUtf8SearchHit(t *testing.T) {
	pattern, err := regexp.Compile("ä")
	if err != nil {
		panic(err)
	}

	line := _CreateScreenLine(nil, 0, 0, 3, "åäö", pattern)
	assertTokenRangesEqual(t, line, []Token{
		_CreateExpectedCell('å', tcell.StyleDefault),
		_CreateExpectedCell('ä', tcell.StyleDefault.Reverse(true)),
		_CreateExpectedCell('ö', tcell.StyleDefault),
	})
}

func TestCreateScreenLineScrolledUtf8SearchHit(t *testing.T) {
	pattern, err := regexp.Compile("ä")
	if err != nil {
		panic(err)
	}

	line := _CreateScreenLine(nil, 0, 1, 4, "ååäö", pattern)
	assertTokenRangesEqual(t, line, []Token{
		_CreateExpectedCell('<', tcell.StyleDefault.Reverse(true)),
		_CreateExpectedCell('å', tcell.StyleDefault),
		_CreateExpectedCell('ä', tcell.StyleDefault.Reverse(true)),
		_CreateExpectedCell('ö', tcell.StyleDefault),
	})
}
