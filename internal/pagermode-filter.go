package internal

import (
	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/twin"
)

type PagerModeFilter struct {
	pager        *Pager
	filterString string
}

func (m PagerModeFilter) drawFooter(_ string, _ string) {
	width, height := m.pager.screen.Size()

	prompt := "Filter: "

	pos := 0
	for _, token := range prompt + m.filterString {
		pos += m.pager.screen.SetCell(pos, height-1, twin.NewStyledRune(token, twin.StyleDefault))
	}

	// Add a cursor
	pos += m.pager.screen.SetCell(pos, height-1, twin.NewStyledRune(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))

	// Clear the rest of the line
	for pos < width {
		pos += m.pager.screen.SetCell(pos, height-1, twin.NewStyledRune(' ', twin.StyleDefault))
	}
}

func (m *PagerModeFilter) onKey(key twin.KeyCode) {
	switch key {
	case twin.KeyEnter:
		m.pager.mode = PagerModeViewing{pager: m.pager}

	case twin.KeyEscape:
		m.pager.mode = PagerModeViewing{pager: m.pager}
		m.pager.filterPattern = nil
		m.pager.searchString = ""
		m.pager.searchPattern = nil

	case twin.KeyBackspace, twin.KeyDelete:
		if len(m.filterString) == 0 {
			return
		}

		m.filterString = removeLastChar(m.filterString)
		m.pager.filterPattern = toPattern(m.filterString)
		m.pager.searchString = m.filterString
		m.pager.searchPattern = toPattern(m.filterString)

	case twin.KeyUp, twin.KeyDown, twin.KeyRight, twin.KeyLeft, twin.KeyPgUp, twin.KeyPgDown, twin.KeyHome, twin.KeyEnd:
		viewing := PagerModeViewing{pager: m.pager}

		// Scroll up / down
		viewing.onKey(key)

	default:
		log.Debugf("Unhandled filter key event %v", key)
	}
}

func (m *PagerModeFilter) onRune(char rune) {
	if char == '\x08' {
		// Backspace
		if len(m.filterString) == 0 {
			return
		}

		m.filterString = removeLastChar(m.filterString)
	} else {
		m.filterString = m.filterString + string(char)
	}

	m.pager.filterPattern = toPattern(m.filterString)
	m.pager.searchString = m.filterString
	m.pager.searchPattern = toPattern(m.filterString)
}
