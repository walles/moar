package m

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell"
)

// Pager is the main on-screen pager
type _Pager struct {
	reader _Reader
}

// NewPager creates a new Pager
func NewPager(r _Reader) *_Pager {
	return &_Pager{
		reader: r,
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

	quit := make(chan struct{})

	s.Clear()
	for pos, char := range "Press ESC / Return to exit" {
		s.SetContent(5+pos, 5, char, nil, tcell.StyleDefault)
	}
	s.Show()

	// Main loop
	go func() {
		for {
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
