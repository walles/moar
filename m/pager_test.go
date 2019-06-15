package m

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell"
)

func TestUnicodeRendering(t *testing.T) {
	reader, err := NewReaderFromStream(strings.NewReader("åäö"))
	if err != nil {
		panic(err)
	}

	screen := tcell.NewSimulationScreen("UTF-8")
	pager := NewPager(*reader)
	pager.Quit()
	pager.StartPaging(screen)

	var answers = []_ExpectedCell{
		_CreateExpectedCell('å', tcell.StyleDefault),
		_CreateExpectedCell('ä', tcell.StyleDefault),
		_CreateExpectedCell('ö', tcell.StyleDefault),
	}

	contents, _, _ := screen.GetContents()
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

	screen := tcell.NewSimulationScreen("UTF-8")
	pager := NewPager(*reader)
	pager.Quit()
	pager.StartPaging(screen)

	var answers = []_ExpectedCell{
		_CreateExpectedCell('a', tcell.StyleDefault.Foreground(tcell.ColorBlack)),
		_CreateExpectedCell('b', tcell.StyleDefault.Foreground(tcell.ColorRed)),
		_CreateExpectedCell('c', tcell.StyleDefault.Foreground(tcell.ColorGreen)),
		_CreateExpectedCell('d', tcell.StyleDefault.Foreground(tcell.ColorYellow)),
		_CreateExpectedCell('e', tcell.StyleDefault.Foreground(tcell.ColorBlue)),
		_CreateExpectedCell('f', tcell.StyleDefault.Foreground(tcell.ColorPurple)),
		_CreateExpectedCell('g', tcell.StyleDefault.Foreground(tcell.ColorTeal)),
		_CreateExpectedCell('h', tcell.StyleDefault.Foreground(tcell.ColorWhite)),
		_CreateExpectedCell('i', tcell.StyleDefault),
	}

	contents, _, _ := screen.GetContents()
	for pos, expected := range answers {
		expected.LogDifference(t, contents[pos])
	}
}
