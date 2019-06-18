package m

import (
	"log"
	"strings"
	"testing"

	"github.com/gdamore/tcell"
)

func TestUnicodeRendering(t *testing.T) {
	reader, err := NewReaderFromStream(strings.NewReader("åäö"))
	if err != nil {
		panic(err)
	}

	var answers = []_ExpectedCell{
		_CreateExpectedCell('å', tcell.StyleDefault),
		_CreateExpectedCell('ä', tcell.StyleDefault),
		_CreateExpectedCell('ö', tcell.StyleDefault),
	}

	contents := _StartPaging(t, reader)
	for pos, expected := range answers {
		expected.LogDifference(t, contents[pos])
	}
}

type _ExpectedCell struct {
	Rune  rune
	Style tcell.Style
}

func (expected _ExpectedCell) LogDifference(t *testing.T, actual tcell.SimCell) {
	if actual.Runes[0] == expected.Rune && actual.Style == expected.Style {
		return
	}

	t.Errorf("Expected '%s'/0x%x, got '%s'/0x%x",
		string(expected.Rune), expected.Style,
		string(actual.Runes[0]), actual.Style)
}

func _CreateExpectedCell(Rune rune, Style tcell.Style) _ExpectedCell {
	return _ExpectedCell{
		Rune:  Rune,
		Style: Style,
	}
}

func TestFgColorRendering(t *testing.T) {
	reader, err := NewReaderFromStream(strings.NewReader(
		"\x1b[30ma\x1b[31mb\x1b[32mc\x1b[33md\x1b[34me\x1b[35mf\x1b[36mg\x1b[37mh\x1b[0mi"))
	if err != nil {
		panic(err)
	}

	var answers = []_ExpectedCell{
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
	reader, err := NewReaderFromStream(strings.NewReader(
		"abc\xc2def"))
	if err != nil {
		panic(err)
	}

	var answers = []_ExpectedCell{
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
	pager := NewPager(*reader)
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
	reader, err := NewReaderFromStream(strings.NewReader(s))
	if err != nil {
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

// FIXME: Add man page formatting tests
