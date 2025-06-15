package m

import (
	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/twin"
)

type PagerModeFilter struct {
	pager *Pager
}

func (m PagerModeFilter) drawFooter(_ string, _ string) {
	width, height := m.pager.screen.Size()

	prompt := "Filter: "

	pos := 0
	for _, token := range prompt + m.pager.searchString {
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
	case twin.KeyEnter, twin.KeyEscape:
		m.pager.mode = PagerModeViewing{pager: m.pager}
		m.pager.searchPattern = nil

	case twin.KeyBackspace, twin.KeyDelete:
		if len(m.pager.searchString) == 0 {
			return
		}

		m.pager.searchString = removeLastChar(m.pager.searchString)
		m.pager.searchPattern = toPattern(m.pager.searchString)

	case twin.KeyUp, twin.KeyDown, twin.KeyPgUp, twin.KeyPgDown:
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
		if len(m.pager.searchString) == 0 {
			return
		}

		m.pager.searchString = removeLastChar(m.pager.searchString)
	} else {
		m.pager.searchString = m.pager.searchString + string(char)
	}

	m.pager.searchPattern = toPattern(m.pager.searchString)
}
