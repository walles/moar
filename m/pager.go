package m

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell"
)

// Pager is the main on-screen pager
type _Pager struct {
	reader _Reader
	screen tcell.Screen
	quit   chan struct{}
}

// NewPager creates a new Pager
func NewPager(r _Reader) *_Pager {
	return &_Pager{
		reader: r,
		quit:   make(chan struct{}),
	}
}

func (p *_Pager) _AddFooter() {
	_, height := p.screen.Size()

	for pos, char := range "Press ESC / Return to exit" {
		p.screen.SetContent(pos, height-1, char, nil, tcell.StyleDefault)
	}
}

func (p *_Pager) _Redraw() {
	p.screen.Clear()

	// FIXME: Ask our reader for lines and draw them

	p._AddFooter()
	p.screen.Sync()
}

func (p *_Pager) _Quit() {
	close(p.quit)
}

func (p *_Pager) _OnKey(key tcell.Key) {
	switch key {
	case tcell.KeyEscape, tcell.KeyEnter:
		p._Quit()
	}
}

// StartPaging brings up the pager on screen
func (p *_Pager) StartPaging() {
	// This function initially inspired by
	// https://github.com/gdamore/tcell/blob/master/_demos/unicode.go
	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	if e = s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	defer s.Fini()

	p.screen = s

	// Main loop
	go func() {
		s.Show()
		for {
			p._Redraw()

			ev := s.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				p._OnKey(ev.Key())

			case *tcell.EventResize:
				// We'll be implicitly redrawn just by taking another lap in the loop
			}
		}
	}()

	<-p.quit
}
