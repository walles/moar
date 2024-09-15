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
		p.screen.SetCell(pos, height-1, twin.NewStyledRune(token, twin.StyleDefault))
		pos++
	}
}

func (m PagerModeJumpToMark) getMarkPrompt() string {
	// Special case having zero, one or multiple marks
	if len(m.pager.marks) == 0 {
		return "No marks set, press 'm' to set one!"
	}

	if len(m.pager.marks) == 1 {
		for key := range m.pager.marks {
			return "Jump to your mark: " + string(key)
		}
	}

	// Multiple marks, list them
	marks := maps.Keys(m.pager.marks)
	sort.Slice(marks, func(i, j int) bool {
		return marks[i] < marks[j]
	})

	prompt := "Jump to one of these marks: "
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
	if len(m.pager.marks) == 0 && char == 'm' {
		//nolint:gosimple // The linter's advice is just wrong here
		m.pager.mode = PagerModeMark{pager: m.pager}
		return
	}

	destination, ok := m.pager.marks[char]
	if ok {
		m.pager.scrollPosition = destination
	}

	//nolint:gosimple // The linter's advice is just wrong here
	m.pager.mode = PagerModeViewing{pager: m.pager}
}
