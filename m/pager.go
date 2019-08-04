package m

import (
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
	"time"
	"unicode"

	"github.com/gdamore/tcell"
)

// FIXME: Profile the pager while searching through a large file

type _PagerMode int

const (
	_Viewing   _PagerMode = 0
	_Searching _PagerMode = 1
	_NotFound  _PagerMode = 2
)

// Pager is the main on-screen pager
type _Pager struct {
	reader              *Reader
	screen              tcell.Screen
	quit                bool
	firstLineOneBased   int
	leftColumnZeroBased int

	mode          _PagerMode
	searchString  string
	searchPattern *regexp.Regexp

	isShowingHelp bool
	preHelpState  *_PreHelpState
}

type _PreHelpState struct {
	reader              *Reader
	firstLineOneBased   int
	leftColumnZeroBased int
}

const _EofMarker = "\x1b[7m---" // Reverse video "---""

var _HelpReader = NewReaderFromText("Help", `
Welcome to Moar, the nice pager!

Quitting
--------
* Press 'q' or ESC to quit

Moving around
-------------
* Arrow keys
* PageUp / 'b' and PageDown / 'f'
* Half page 'u'p / 'd'own
* Home and End for start / end of the document
* < to go to the start of the document
* > to go to the end of the document
* RETURN moves down one line
* SPACE moves down a page

Searching
---------
* Type / to start searching, then type what you want to find
* Type RETURN to stop searching
* Find next by typing 'n' (for "next")
* Find previous by typing SHIFT-N or 'p' (for "previous")
* Search is case sensitive if it contains any UPPER CASE CHARACTERS
* Search is interpreted as a regexp if it is a valid one

Reporting bugs
--------------
File issues at https://github.com/walles/moar/issues, or post
questions to johan.walles@gmail.com.

Installing Moar as your default pager
-------------------------------------
Put the following line in your .bashrc or .bash_profile:
  export PAGER=/usr/local/bin/moar.rb

Source Code
-----------
Available at https://github.com/walles/moar/.
`)

// NewPager creates a new Pager
func NewPager(r *Reader) *_Pager {
	return &_Pager{
		reader:            r,
		quit:              false,
		firstLineOneBased: 1,
	}
}

func (p *_Pager) _AddLine(logger *log.Logger, lineNumber int, line string) {
	pos := 0
	stringIndexAtColumnZero := p.leftColumnZeroBased
	if p.leftColumnZeroBased > 0 {
		// Indicate that it's possible to scroll left
		p.screen.SetContent(pos, lineNumber, '<', nil, tcell.StyleDefault.Reverse(true))
		pos++

		// This code can be verified by searching for "monkeys" in
		// sample-files/long-and-wide.txt and scrolling right. If the
		// "monkeys" highlight is in the right place both before and
		// after scrolling right then this code is good.
		stringIndexAtColumnZero--
	}

	tokens, plainString := TokensFromString(logger, line)
	if p.leftColumnZeroBased >= len(tokens) {
		// Nothing to display, never mind
		return
	}

	matchRanges := GetMatchRanges(plainString, p.searchPattern)
	for _, token := range tokens[p.leftColumnZeroBased:] {
		width, _ := p.screen.Size()
		if pos >= width {
			// Indicate that this line continues to the right
			p.screen.SetContent(pos-1, lineNumber, '>', nil, tcell.StyleDefault.Reverse(true))
			break
		}

		style := token.Style
		if matchRanges.InRange(pos + stringIndexAtColumnZero) {
			// Search hits in reverse video
			style = style.Reverse(true)
		}

		p.screen.SetContent(pos, lineNumber, token.Rune, nil, style)

		pos++
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

	screenLineNumber := 0
	for _, line := range lines.lines {
		p._AddLine(logger, screenLineNumber, line)
		screenLineNumber++
	}

	p._AddLine(logger, screenLineNumber, _EofMarker)

	switch p.mode {
	case _Searching:
		p._AddSearchFooter()

	case _NotFound:
		p._SetFooter("Not found: " + p.searchString)

	case _Viewing:
		helpText := "Press ESC / q to exit, '/' to search, 'h' for help"
		if p.isShowingHelp {
			helpText = "Press ESC / q to exit help, '/' to search"
		}
		p._SetFooter(lines.statusText + "  " + helpText)

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
	if !p.isShowingHelp {
		p.quit = true
		return
	}

	// Reset help
	p.isShowingHelp = false
	p.reader = p.preHelpState.reader
	p.firstLineOneBased = p.preHelpState.firstLineOneBased
	p.leftColumnZeroBased = p.preHelpState.leftColumnZeroBased
	p.preHelpState = nil
}

func (p *_Pager) _ScrollToSearchHits() {
	if p.searchPattern == nil {
		// This is not a search
		return
	}

	firstHitLine := p._FindFirstHitLineOneBased(p.firstLineOneBased, false)
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

func (p *_Pager) _FindFirstHitLineOneBased(firstLineOneBased int, backwards bool) *int {
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

		if backwards {
			lineNumber--
		} else {
			lineNumber++
		}
	}
}

func (p *_Pager) _ScrollToNextSearchHit() {
	if p.searchPattern == nil {
		// Nothing to search for, never mind
		return
	}

	if p.reader.GetLineCount() == 0 {
		// Nothing to search in, never mind
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

	firstHitLine := p._FindFirstHitLineOneBased(firstSearchLineOneBased, false)
	if firstHitLine == nil {
		p.mode = _NotFound
		return
	}
	p.firstLineOneBased = *firstHitLine
}

func (p *_Pager) _ScrollToPreviousSearchHit() {
	if p.searchPattern == nil {
		// Nothing to search for, never mind
		return
	}

	if p.reader.GetLineCount() == 0 {
		// Nothing to search in, never mind
		return
	}

	var firstSearchLineOneBased int

	switch p.mode {
	case _Viewing:
		// Start searching on the first line above the top of the screen
		firstSearchLineOneBased = p.firstLineOneBased - 1

	case _NotFound:
		// Restart searching from the bottom
		p.mode = _Viewing
		firstSearchLineOneBased = p.reader.GetLineCount()

	default:
		panic(fmt.Sprint("Unknown search mode when finding previous: ", p.mode))
	}

	firstHitLine := p._FindFirstHitLineOneBased(firstSearchLineOneBased, true)
	if firstHitLine == nil {
		p.mode = _NotFound
		return
	}
	p.firstLineOneBased = *firstHitLine
}

func (p *_Pager) _UpdateSearchPattern() {
	p.searchPattern = ToPattern(p.searchString)

	p._ScrollToSearchHits()

	// FIXME: If the user is typing, indicate to user if we didn't find anything
}

// ToPattern compiles a search string into a pattern.
//
// If the string contains only lower-case letter the pattern will be case insensitive.
//
// If the string is empty the pattern will be nil.
//
// If the string does not compile into a regexp the pattern will match the string verbatim
func ToPattern(compileMe string) *regexp.Regexp {
	if len(compileMe) == 0 {
		return nil
	}

	hasUppercase := false
	for _, char := range compileMe {
		if unicode.IsUpper(char) {
			hasUppercase = true
		}
	}

	// Smart case; be case insensitive unless there are upper case chars
	// in the search string
	prefix := "(?i)"
	if hasUppercase {
		prefix = ""
	}

	pattern, err := regexp.Compile(prefix + compileMe)
	if err == nil {
		// Search string is a regexp
		return pattern
	}

	pattern, err = regexp.Compile(prefix + regexp.QuoteMeta(compileMe))
	if err == nil {
		// Pattern matching the string exactly
		return pattern
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

	case tcell.KeyUp:
		// Clipping is done in _AddLines()
		p.firstLineOneBased--
		p.mode = _Viewing

	case tcell.KeyDown:
		// Clipping is done in _AddLines()
		p.firstLineOneBased++
		p.mode = _Viewing

	case tcell.KeyPgUp:
		_, height := p.screen.Size()
		p.firstLineOneBased -= (height - 1)
		p.mode = _Viewing

	case tcell.KeyPgDn:
		_, height := p.screen.Size()
		p.firstLineOneBased += (height - 1)
		p.mode = _Viewing

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

	switch key {
	case tcell.KeyEscape:
		p.Quit()

	case tcell.KeyUp:
		// Clipping is done in _AddLines()
		p.firstLineOneBased--

	case tcell.KeyDown, tcell.KeyEnter:
		// Clipping is done in _AddLines()
		p.firstLineOneBased++

	case tcell.KeyRight:
		p.leftColumnZeroBased += 16

	case tcell.KeyLeft:
		p.leftColumnZeroBased -= 16
		if p.leftColumnZeroBased < 0 {
			p.leftColumnZeroBased = 0
		}

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

	case 'h':
		if !p.isShowingHelp {
			p.preHelpState = &_PreHelpState{
				reader:              p.reader,
				firstLineOneBased:   p.firstLineOneBased,
				leftColumnZeroBased: p.leftColumnZeroBased,
			}
			p.reader = _HelpReader
			p.firstLineOneBased = 1
			p.leftColumnZeroBased = 0
			p.isShowingHelp = true
		}

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

	case 'u':
		_, height := p.screen.Size()
		p.firstLineOneBased -= (height / 2)

	case 'd':
		_, height := p.screen.Size()
		p.firstLineOneBased += (height / 2)

	case '/':
		p.mode = _Searching
		p.searchString = ""
		p.searchPattern = nil

	case 'n':
		p._ScrollToNextSearchHit()

	case 'p', 'N':
		p._ScrollToPreviousSearchHit()

	default:
		logger.Printf("Unhandled rune keypress '%s'", string(char))
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

	go func() {
		for {
			// Wait for new lines to appear
			<-p.reader.moreLinesAdded
			screen.PostEvent(tcell.NewEventInterrupt(nil))

			// Delay updates a bit so that we don't waste time refreshing
			// the screen too often.
			//
			// Note that the delay is *after* reacting, this way single-line
			// updates are reacted to immediately, and the first output line
			// read will appear on screen without delay.
			time.Sleep(200 * time.Millisecond)
		}
	}()

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

		case *tcell.EventInterrupt:
			// This means we got more lines, look for NewEventInterrupt higher up
			// in this file. Doing nothing here is fine, the refresh happens after
			// this switch statement.

		default:
			logger.Printf("Unhandled event type: %v", ev)
		}

		// FIXME: If more events are ready, skip this redraw, that
		// should speed up mouse wheel scrolling

		p._Redraw(logger)
	}

	// FIXME: Log p.reader.err if it's non-nil
}
