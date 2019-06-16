package m

import (
	"fmt"
	"log"
	"os"

	"github.com/gdamore/tcell"
)

// Pager is the main on-screen pager
type _Pager struct {
	reader            Reader
	screen            tcell.Screen
	quit              bool
	firstLineOneBased int
}

// NewPager creates a new Pager
func NewPager(r Reader) *_Pager {
	return &_Pager{
		reader:            r,
		quit:              false,
		firstLineOneBased: 1,
	}
}

func (p *_Pager) _AddLine(lineNumber int, line string) {
	for pos, token := range TokensFromString(line) {
		p.screen.SetContent(pos, lineNumber, token.Rune, nil, token.Style)
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
	width, height := p.screen.Size()

	footerText := "Press ESC / q to exit"

	reverse := tcell.Style.Reverse(tcell.StyleDefault, true)
	for pos, char := range footerText {
		p.screen.SetContent(pos, height-1, char, nil, reverse)
	}

	for pos := len(footerText); pos < width; pos++ {
		p.screen.SetContent(pos, height-1, ' ', nil, reverse)
	}
}

func (p *_Pager) _Redraw() {
	p.screen.Clear()

	p._AddLines()

	p._AddFooter()
	p.screen.Sync()
}

func (p *_Pager) Quit() {
	p.quit = true
}

func (p *_Pager) _OnKey(logger *log.Logger, key tcell.Key) {
	switch key {
	case tcell.KeyEscape:
		p.Quit()

	case tcell.KeyUp:
		// Clipping is done in _AddLines()
		p.firstLineOneBased--

	case tcell.KeyDown, tcell.KeyEnter:
		// Clipping is done in _AddLines()
		p.firstLineOneBased++

	case tcell.KeyHome:
		p.firstLineOneBased = 1

	case tcell.KeyEnd:
		p.firstLineOneBased = p.reader.LineCount() + 1

	case tcell.KeyPgDn:
		_, height := p.screen.Size()
		p.firstLineOneBased += (height - 1)

	case tcell.KeyPgUp:
		_, height := p.screen.Size()
		p.firstLineOneBased -= (height - 1)

	default:
		logger.Printf("Unhandled rune key event %v", key)
	}
}

func (p *_Pager) _OnRune(logger *log.Logger, char rune) {
	switch char {
	case 'q':
		p.Quit()

	case 'k', 'y':
		// Clipping is done in _AddLines()
		p.firstLineOneBased--

	case 'j', 'e':
		// Clipping is done in _AddLines()
		p.firstLineOneBased++

	case '<', 'g':
		p.firstLineOneBased = 1

	case '>', 'G':
		p.firstLineOneBased = p.reader.LineCount() + 1

	case 'f', ' ':
		_, height := p.screen.Size()
		p.firstLineOneBased += (height - 1)

	case 'b':
		_, height := p.screen.Size()
		p.firstLineOneBased -= (height - 1)

	default:
		logger.Printf("Unhandled rune keyress '%s'", string(char))
	}
}

// StartPaging brings up the pager on screen
func (p *_Pager) StartPaging(logger *log.Logger, screen tcell.Screen) {
	// This function initially inspired by
	// https://github.com/gdamore/tcell/blob/master/_demos/unicode.go
	if e := screen.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	p.screen = screen
	screen.Show()
	p._Redraw()

	// Main loop
	for !p.quit {
		ev := screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyRune {
				p._OnRune(logger, ev.Rune())
			} else {
				p._OnKey(logger, ev.Key())
			}

		case *tcell.EventResize:
			// We'll be implicitly redrawn just by taking another lap in the loop

		default:
			logger.Printf("Unhandled event type: %v", ev)
		}

		p._Redraw()
	}
}
