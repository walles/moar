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
	switch key {
	case twin.KeyEnter:
		newLineNumber, err := strconv.Atoi(p.gotoLineString)
		if err == nil {
			p.scrollPosition = NewScrollPositionFromLineNumberOneBased(newLineNumber, "onGotoLineKey")
		}
		p.mode = _Viewing

	case twin.KeyEscape:
		p.mode = _Viewing

	case twin.KeyBackspace, twin.KeyDelete:
		if len(p.gotoLineString) == 0 {
			return
		}

		p.gotoLineString = removeLastChar(p.gotoLineString)

	default:
		log.Tracef("Unhandled goto key event %v, treating as a viewing key event", key)
		p.mode = _Viewing
		p.onKey(key)
	}
}

func (p *Pager) onGotoLineRune(char rune) {
	if char == 'q' {
		p.mode = _Viewing
		return
	}

	newGotoLineString := p.gotoLineString + string(char)
	_, err := strconv.Atoi(newGotoLineString)
	if err != nil {
		log.Debugf("Got non-number goto rune '%s'/0x%08x: %s", string(char), int32(char), err)
		return
	}

	p.gotoLineString = newGotoLineString
}
