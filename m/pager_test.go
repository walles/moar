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
		_CreateExpectedCell('a', tcell.StyleDefault.Foreground(tcell.ColorBlack)),
		_CreateExpectedCell('b', tcell.StyleDefault.Foreground(tcell.ColorLightCoral)),
		_CreateExpectedCell('c', tcell.StyleDefault.Foreground(tcell.ColorLightGreen)),
		_CreateExpectedCell('d', tcell.StyleDefault.Foreground(tcell.ColorLightYellow)),
		_CreateExpectedCell('e', tcell.StyleDefault.Foreground(tcell.ColorLightBlue)),
		_CreateExpectedCell('f', tcell.StyleDefault.Foreground(tcell.ColorPink)),
		_CreateExpectedCell('g', tcell.StyleDefault.Foreground(tcell.ColorLightCyan)),
		_CreateExpectedCell('h', tcell.StyleDefault.Foreground(tcell.ColorWhite)),
		_CreateExpectedCell('i', tcell.StyleDefault),
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

// FIXME: Add background color tests

// FIXME: Add tests for various forms of highlighting

// FIXME: Add man page formatting tests

// FIXME: Add TAB handling tests
