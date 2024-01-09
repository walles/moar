package m

import "github.com/walles/moar/twin"

type PagerModeNotFound struct {
	pager *Pager
}

func (m PagerModeNotFound) drawFooter(_ string, _ string) {
	m.pager.setFooter("Not found: " + m.pager.searchString)
}

func (m PagerModeNotFound) onKey(key twin.KeyCode) {
	//nolint:gosimple // The linter's advice is just wrong here
	m.pager.mode = PagerModeViewing{pager: m.pager}
	m.pager.mode.onKey(key)
}

func (m PagerModeNotFound) onRune(char rune) {
	switch char {

	// Should match the pagermode-viewing.go next-search-hit bindings
	case 'n':
		m.pager.scrollToNextSearchHit()

	// Should match the pagermode-viewing.go previous-search-hit bindings
	case 'p', 'N':
		m.pager.scrollToPreviousSearchHit()

	default:
		//nolint:gosimple // The linter's advice is just wrong here
		m.pager.mode = PagerModeViewing{pager: m.pager}
		m.pager.mode.onRune(char)
	}
}
