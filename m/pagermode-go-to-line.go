package m

import (
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/twin"
)

type PagerModeGotoLine struct {
	pager *Pager
}

func (m PagerModeGotoLine) drawFooter(statusText string, spinner string) {
	p := m.pager

	_, height := p.screen.Size()

	pos := 0
	for _, token := range "Go to line number: " + p.gotoLineString {
		p.screen.SetCell(pos, height-1, twin.NewCell(token, twin.StyleDefault))
		pos++
	}

	// Add a cursor
	p.screen.SetCell(pos, height-1, twin.NewCell(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))
}

func (m PagerModeGotoLine) onKey(key twin.KeyCode) {
	p := m.pager

	switch key {
	case twin.KeyEnter:
		newLineNumber, err := strconv.Atoi(p.gotoLineString)
		if err == nil {
			p.scrollPosition = NewScrollPositionFromLineNumber(
				linenumbers.LineNumberFromOneBased(newLineNumber),
				"onGotoLineKey",
			)
		}
		p.mode = PagerModeViewing{pager: p}

	case twin.KeyEscape:
		p.mode = PagerModeViewing{pager: p}

	case twin.KeyBackspace, twin.KeyDelete:
		if len(p.gotoLineString) == 0 {
			return
		}

		p.gotoLineString = removeLastChar(p.gotoLineString)

	default:
		log.Tracef("Unhandled goto key event %v, treating as a viewing key event", key)
		p.mode = PagerModeViewing{pager: p}
		p.mode.onKey(key)
	}
}

func (m PagerModeGotoLine) onRune(char rune) {
	p := m.pager

	if char == 'q' {
		p.mode = PagerModeViewing{pager: p}
		return
	}

	if char == 'g' {
		p.scrollPosition = newScrollPosition("Pager scroll position")
		p.handleScrolledUp()
		p.mode = PagerModeViewing{pager: p}
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
