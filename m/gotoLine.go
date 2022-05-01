package m

import (
	"github.com/walles/moar/twin"
)

func (p *Pager) addGotoLineFooter() {
	_, height := p.screen.Size()

	pos := 0
	for _, token := range "Go to line number: " + p.gotoLineString {
		p.screen.SetCell(pos, height-1, twin.NewCell(token, twin.StyleDefault))
		pos++
	}

	// Add a cursor
	p.screen.SetCell(pos, height-1, twin.NewCell(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))
}

func (p *Pager) onGotoLineKey(key twin.KeyCode) {
	FIXME() // Implement based on onSearchKey
}

func (p *Pager) onGotoLineRune(char rune) {
	FIXME() // Implement based on onSearchRune
}
