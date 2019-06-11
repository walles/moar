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
}

// NewPager creates a new Pager
func NewPager(r _Reader) *_Pager {
	return &_Pager{
		reader: r,
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

// StartPaging brings up the pager on screen
func (p *_Pager) StartPaging() {
	// This function initially inspired by
	// https://github.com/gdamore/tcell/blob/master/_demos/unicode.go
	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	p.screen = s

	if e = s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	quit := make(chan struct{})

	// Main loop
	go func() {
		s.Show()
		for {
			p._Redraw()

			ev := s.PollEvent()
			switch ev := ev.(type) {
			case *tcell.EventKey:
				switch ev.Key() {
				case tcell.KeyEscape, tcell.KeyEnter:
					close(quit)
					return
				case tcell.KeyCtrlL:
					s.Sync()
				}
			case *tcell.EventResize:
				s.Sync()
			}
		}
	}()

	<-quit

	s.Fini()
}
