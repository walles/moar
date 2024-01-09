package m

import "github.com/walles/moar/twin"

type PagerModeMark struct {
	pager *Pager
}

func (m PagerModeMark) drawFooter(_ string, _ string) {
	p := m.pager

	_, height := p.screen.Size()

	pos := 0
	for _, token := range "Press any key to label your mark: " {
		p.screen.SetCell(pos, height-1, twin.NewCell(token, twin.StyleDefault))
		pos++
	}

	// Add a cursor
	p.screen.SetCell(pos, height-1, twin.NewCell(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))
}

func (m PagerModeMark) onKey(key twin.KeyCode) {
	p := m.pager

	switch key {
	case twin.KeyEnter, twin.KeyEscape:
		// Never mind I
		p.mode = PagerModeViewing{pager: p}

	default:
		// Never mind II
		p.mode = PagerModeViewing{pager: p}
		p.mode.onKey(key)
	}
}

func (m PagerModeMark) onRune(char rune) {
	m.pager.marks[char] = m.pager.scrollPosition

	//nolint:gosimple // The linter's advice is just wrong here
	m.pager.mode = PagerModeViewing{pager: m.pager}
}
