package m

import (
	"fmt"
	"log"
	"math"
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

func (p *_Pager) _AddLine(logger *log.Logger, lineNumber int, line string) {
	for pos, token := range TokensFromString(logger, line) {
		p.screen.SetContent(pos, lineNumber, token.Rune, nil, token.Style)
	}
}

func (p *_Pager) _AddLines(logger *log.Logger) {
	width, height := p.screen.Size()
	wantedLineCount := height - 1

	lines := p.reader.GetLines(p.firstLineOneBased, wantedLineCount)

	// If we're asking for past-the-end lines, the Reader will clip for us,
	// and we should adapt to that. Otherwise if you scroll 100 lines past
	// the end, you'll then have to scroll 100 lines up again before the
	// display starts scrolling visibly.
	p.firstLineOneBased = lines.firstLineOneBased

	for screenLineNumber, line := range lines.lines {
		p._AddLine(logger, screenLineNumber, line)
	}

	pos := 0
	footerStyle := tcell.StyleDefault.Reverse(true)
	for _, token := range "Press ESC / q to exit" {
		p.screen.SetContent(pos, height-1, token, nil, footerStyle)
		pos++
	}

	for ; pos < width; pos++ {
		p.screen.SetContent(pos, height-1, ' ', nil, footerStyle)
	}
}

func (p *_Pager) _Redraw(logger *log.Logger) {
	p.screen.Clear()

	p._AddLines(logger)

	p.screen.Show()
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
		p.firstLineOneBased = math.MaxInt32

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
		p.firstLineOneBased = math.MaxInt32

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
	// We want to match the terminal theme, see screen.Init() source code
	os.Setenv("TCELL_TRUECOLOR", "disable")

	if e := screen.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}

	p.screen = screen
	screen.Show()
	p._Redraw(logger)

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

		p._Redraw(logger)
	}
}
