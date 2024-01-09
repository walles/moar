package m

import (
	"sort"

	"github.com/walles/moar/twin"
	"golang.org/x/exp/maps"
)

type PagerModeJumpToMark struct {
	pager *Pager
}

func (m PagerModeJumpToMark) drawFooter(_ string, _ string) {
	p := m.pager

	_, height := p.screen.Size()

	pos := 0
	for _, token := range m.getMarkPrompt() {
		p.screen.SetCell(pos, height-1, twin.NewCell(token, twin.StyleDefault))
		pos++
	}

	// Add a cursor
	p.screen.SetCell(pos, height-1, twin.NewCell(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))
}

func (m PagerModeJumpToMark) getMarkPrompt() string {
	// Special case having zero, one or multiple marks
	if len(m.pager.marks) == 0 {
		return "Press \"m\" to set your first mark!"
	}

	if len(m.pager.marks) == 1 {
		for key := range m.pager.marks {
			return "Press \"" + string(key) + "\" to jump to your mark!"
		}
	}

	// Multiple marks, list them
	marks := maps.Keys(m.pager.marks)
	sort.Slice(marks, func(i, j int) bool {
		return marks[i] < marks[j]
	})

	prompt := "Press a key to jump to your mark: "
	for i, mark := range marks {
		if i > 0 {
			prompt += ", "
		}
		prompt += string(mark)
	}

	return prompt
}

func (m PagerModeJumpToMark) onKey(key twin.KeyCode) {
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

func (m PagerModeJumpToMark) onRune(char rune) {
	destination, ok := m.pager.marks[char]
	if ok {
		m.pager.scrollPosition = destination
	}

	//nolint:gosimple // The linter's advice is just wrong here
	m.pager.mode = PagerModeViewing{pager: m.pager}
}
