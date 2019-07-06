package m

import (
	"fmt"
	"log"
	"math"
	"os"
	"regexp"

	"github.com/gdamore/tcell"
)

type _PagerMode int

const (
	_Viewing   _PagerMode = 0
	_Searching _PagerMode = 1
	_NotFound  _PagerMode = 2
)

// Pager is the main on-screen pager
type _Pager struct {
	reader            Reader
	screen            tcell.Screen
	quit              bool
	firstLineOneBased int

	mode          _PagerMode
	searchString  string
	searchPattern *regexp.Regexp
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
	tokens, plainString := TokensFromString(logger, line)
	matchRanges := GetMatchRanges(plainString, p.searchPattern)

	for pos, token := range tokens {
		style := token.Style
		if matchRanges.InRange(pos) {
			// FIXME: This doesn't work if the style is already reversed
			style = style.Reverse(true)
		}

		p.screen.SetContent(pos, lineNumber, token.Rune, nil, style)
	}
}

func (p *_Pager) _AddSearchFooter() {
	_, height := p.screen.Size()

	pos := 0
	for _, token := range "Search: " + p.searchString {
		p.screen.SetContent(pos, height-1, token, nil, tcell.StyleDefault)
		pos++
	}

	// Add a cursor
	p.screen.SetContent(pos, height-1, ' ', nil, tcell.StyleDefault.Reverse(true))
}

func (p *_Pager) _AddLines(logger *log.Logger) {
	_, height := p.screen.Size()
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

	switch p.mode {
	case _Searching:
		p._AddSearchFooter()

	case _NotFound:
		p._SetFooter("Not found: " + p.searchString)

	case _Viewing:
		p._SetFooter(lines.statusText + "  Press ESC / q to exit, '/' to search")

	default:
		panic(fmt.Sprint("Unsupported pager mode: ", p.mode))
	}
}

func (p *_Pager) _SetFooter(footer string) {
	width, height := p.screen.Size()

	pos := 0
	footerStyle := tcell.StyleDefault.Reverse(true)
	for _, token := range footer {
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

func (p *_Pager) _ScrollToSearchHits() {
	if p.searchPattern == nil {
		// This is not a search
		return
	}

	firstHitLine := p._FindFirstHitLineOneBased(p.firstLineOneBased)
	if firstHitLine == nil {
		// No match, give up
		return
	}

	if *firstHitLine <= p._GetLastVisibleLineOneBased() {
		// Already on-screen, never mind
		return
	}

	p.firstLineOneBased = *firstHitLine
}

func (p *_Pager) _GetLastVisibleLineOneBased() int {
	firstVisibleLineOneBased := p.firstLineOneBased
	_, windowHeight := p.screen.Size()

	// If first line is 1 and window is 2 high, and one line is the status
	// line, the last line will be 1 + 2 - 2 = 1
	return firstVisibleLineOneBased + windowHeight - 2
}

func (p *_Pager) _FindFirstHitLineOneBased(firstLineOneBased int) *int {
	lineNumber := firstLineOneBased
	for {
		line := p.reader.GetLine(lineNumber)
		if line == nil {
			// No match, give up
			return nil
		}

		if p.searchPattern.MatchString(*line) {
			return &lineNumber
		}

		lineNumber++
	}
}

func (p *_Pager) _ScrollToNextSearchHit() {
	if p.searchPattern == nil {
		// Nothing to search for, never mind
		return
	}

	var firstSearchLineOneBased int

	switch p.mode {
	case _Viewing:
		// Start searching on the first line below the bottom of the screen
		firstSearchLineOneBased = p._GetLastVisibleLineOneBased() + 1

	case _NotFound:
		// Restart searching from the top
		p.mode = _Viewing
		firstSearchLineOneBased = 1

	default:
		panic(fmt.Sprint("Unknown search mode when finding next: ", p.mode))
	}

	firstHitLine := p._FindFirstHitLineOneBased(firstSearchLineOneBased)
	if firstHitLine == nil {
		p.mode = _NotFound
		return
	}
	p.firstLineOneBased = *firstHitLine
}

func (p *_Pager) _UpdateSearchPattern() {
	if len(p.searchString) == 0 {
		p.searchPattern = nil
		return
	}

	defer p._ScrollToSearchHits()
	// FIXME: Indicate to user if we didn't find anything

	pattern, err := regexp.Compile(p.searchString)
	if err == nil {
		// Search string is a regexp
		// FIXME: Make this case insensitive if input is all-lowercase
		p.searchPattern = pattern
		return
	}

	pattern, err = regexp.Compile(regexp.QuoteMeta(p.searchString))
	if err == nil {
		// Pattern matching the string exactly
		// FIXME: Make this case insensitive if input is all-lowercase
		p.searchPattern = pattern
		return
	}

	// Unable to create a match-string-verbatim pattern
	panic(err)
}

func (p *_Pager) _OnSearchKey(logger *log.Logger, key tcell.Key) {
	switch key {
	case tcell.KeyEscape, tcell.KeyEnter:
		p.mode = _Viewing

	case tcell.KeyBackspace, tcell.KeyDEL:
		if len(p.searchString) == 0 {
			return
		}

		p.searchString = p.searchString[:len(p.searchString)-1]
		p._UpdateSearchPattern()

	default:
		logger.Printf("Unhandled search key event %v", key)
	}
}

func (p *_Pager) _OnKey(logger *log.Logger, key tcell.Key) {
	if p.mode == _Searching {
		p._OnSearchKey(logger, key)
		return
	}
	if p.mode != _Viewing && p.mode != _NotFound {
		panic(fmt.Sprint("Unhandled mode: ", p.mode))
	}

	// Reset the not-found marker on non-search keypresses
	p.mode = _Viewing

	// FIXME: Add support for pressing 'h' to get a list of keybindings
	switch key {
	case tcell.KeyEscape:
		p.Quit()

	case tcell.KeyUp:
		// Clipping is done in _AddLines()
		p.firstLineOneBased--

	case tcell.KeyDown, tcell.KeyEnter:
		// Clipping is done in _AddLines()
		p.firstLineOneBased++

	// FIXME: KeyRight and KeyLeft should scroll right and left

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
		logger.Printf("Unhandled key event %v", key)
	}
}

func (p *_Pager) _OnSearchRune(logger *log.Logger, char rune) {
	p.searchString = p.searchString + string(char)
	p._UpdateSearchPattern()
}

func (p *_Pager) _OnRune(logger *log.Logger, char rune) {
	if p.mode == _Searching {
		p._OnSearchRune(logger, char)
		return
	}
	if p.mode != _Viewing && p.mode != _NotFound {
		panic(fmt.Sprint("Unhandled mode: ", p.mode))
	}

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

	case '/':
		p.mode = _Searching
		p.searchString = ""
		p.searchPattern = nil

	case 'n':
		p._ScrollToNextSearchHit()

	// FIXME: Go to previous hit wit 'p'

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

		// FIXME: If more events are ready, skip this redraw, that
		// should speed up mouse wheel scrolling

		p._Redraw(logger)
	}
}
