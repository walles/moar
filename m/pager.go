package m

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell"
)

// Pager is the main on-screen pager
type _Pager struct {
	reader            _Reader
	screen            tcell.Screen
	quit              chan struct{}
	firstLineOneBased int
}

// NewPager creates a new Pager
func NewPager(r _Reader) *_Pager {
	return &_Pager{
		reader:            r,
		quit:              make(chan struct{}),
		firstLineOneBased: 1,
	}
}

func (p *_Pager) _AddLine(lineNumber int, line string) {
	for pos, char := range line {
		p.screen.SetContent(pos, lineNumber, char, nil, tcell.StyleDefault)
	}
}

func (p *_Pager) _AddLines() {
	_, height := p.screen.Size()
	wantedLineCount := height - 1

	lines := p.reader.GetLines(p.firstLineOneBased, wantedLineCount)

	// If we're asking for past-the-end lines, the Reader will clip for us,
	// and we should adapt to that. Otherwise if you scroll 100 lines past
	// the end, you'll then have to scroll 100 lines up again before the
	// display starts scrolling visibly.
	p.firstLineOneBased = lines.firstLineOneBased

	for screenLineNumber, line := range lines.lines {
		p._AddLine(screenLineNumber, line)
	}
}

func (p *_Pager) _AddFooter() {
	_, height := p.screen.Size()
	p._AddLine(height-1, "Press ESC / Return / q to exit")
}

func (p *_Pager) _Redraw() {
	p.screen.Clear()

	p._AddLines()

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

	case tcell.KeyUp:
		// Clipping is done in _AddLines()
		p.firstLineOneBased--

	case tcell.KeyDown:
		// Clipping is done in _AddLines()
		p.firstLineOneBased++
	}
}

func (p *_Pager) _OnRune(char rune) {
	switch char {
	case 'q':
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
				if ev.Key() == tcell.KeyRune {
					p._OnRune(ev.Rune())
				} else {
					p._OnKey(ev.Key())
				}

			case *tcell.EventResize:
				// We'll be implicitly redrawn just by taking another lap in the loop
			}
		}
	}()

	<-p.quit
}
