package m

import (
	"strconv"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/twin"
)

type PagerModeGotoLine struct {
	pager *Pager

	gotoLineString string
}

func (m *PagerModeGotoLine) drawFooter(_ string, _ string) {
	p := m.pager

	_, height := p.screen.Size()

	pos := 0
	for _, token := range "Go to line number: " + m.gotoLineString {
		p.screen.SetCell(pos, height-1, twin.NewStyledRune(token, twin.StyleDefault))
		pos++
	}

	// Add a cursor
	p.screen.SetCell(pos, height-1, twin.NewStyledRune(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))
}

func (m *PagerModeGotoLine) onKey(key twin.KeyCode) {
	p := m.pager

	switch key {
	case twin.KeyEnter:
		newLineNumber, err := strconv.Atoi(m.gotoLineString)
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
		if len(m.gotoLineString) == 0 {
			return
		}

		m.gotoLineString = removeLastChar(m.gotoLineString)

	default:
		log.Tracef("Unhandled goto key event %v, treating as a viewing key event", key)
		p.mode = PagerModeViewing{pager: p}
		p.mode.onKey(key)
	}
}

func (m *PagerModeGotoLine) onRune(char rune) {
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

	newGotoLineString := m.gotoLineString + string(char)
	newGotoLineNumber, err := strconv.Atoi(newGotoLineString)
	if err != nil {
		log.Debugf("Got non-number goto rune '%s'/0x%08x: %s", string(char), int32(char), err)
		return
	}
	if newGotoLineNumber < 1 {
		log.Debugf("Got non-positive goto line number: %d", newGotoLineNumber)
		return
	}

	m.gotoLineString = newGotoLineString
}
