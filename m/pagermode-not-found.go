package m

import "github.com/walles/moar/twin"

type PagerModeNotFound struct {
	pager *Pager
}

func (m PagerModeNotFound) drawFooter(_ string, _ string) {
	m.pager.setFooter("Not found: " + m.pager.searchString)
}

func (m PagerModeNotFound) onKey(key twin.KeyCode) {
	m.pager.mode = PagerModeViewing{pager: m.pager}
	m.pager.mode.onKey(key)
}

func (m PagerModeNotFound) onRune(r rune) {
	m.pager.mode = PagerModeViewing{pager: m.pager}
	m.pager.mode.onRune(r)
}
