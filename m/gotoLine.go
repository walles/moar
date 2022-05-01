package m

import (
	"strconv"

	log "github.com/sirupsen/logrus"
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
	newGotoLineString := p.gotoLineString
	_, err := strconv.Atoi(newGotoLineString)
	if err != nil {
		log.Debugf("Got a non-number rune '%s'", string(char))
		return
	}

	p.gotoLineString = newGotoLineString
}
