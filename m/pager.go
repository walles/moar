package m

import (
	"fmt"
	"regexp"
	"time"
	"unicode"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/twin"
)

type _PagerMode int

const (
	_Viewing   _PagerMode = 0
	_Searching _PagerMode = 1
	_NotFound  _PagerMode = 2
)

type StatusBarStyle int

const (
	STATUSBAR_STYLE_INVERSE StatusBarStyle = iota
	STATUSBAR_STYLE_PLAIN
	STATUSBAR_STYLE_BOLD
)

// How do we render unprintable characters?
type UnprintableStyle int

const (
	UNPRINTABLE_STYLE_HIGHLIGHT UnprintableStyle = iota
	UNPRINTABLE_STYLE_WHITESPACE
)

type eventSpinnerUpdate struct {
	spinner string
}

type eventMoreLinesAvailable struct{}

// Styling of line numbers
var _numberStyle = twin.StyleDefault.WithAttr(twin.AttrDim)

// Pager is the main on-screen pager
type Pager struct {
	reader              *Reader
	screen              twin.Screen
	quit                bool
	firstLineOneBased   int
	leftColumnZeroBased int

	mode          _PagerMode
	searchString  string
	searchPattern *regexp.Regexp

	isShowingHelp bool
	preHelpState  *_PreHelpState

	// NewPager shows lines by default, this field can hide them
	ShowLineNumbers bool

	StatusBarStyle   StatusBarStyle
	UnprintableStyle UnprintableStyle

	WrapLongLines bool

	// If true, pager will clear the screen on return. If false, pager will
	// clear the last line, and show the cursor.
	DeInit bool
}

type _PreHelpState struct {
	reader              *Reader
	firstLineOneBased   int
	leftColumnZeroBased int
}

const _EofMarkerFormat = "\x1b[7m" // Reverse video

var _HelpReader = NewReaderFromText("Help", `
Welcome to Moar, the nice pager!

Miscellaneous
-------------
* Press 'q' or ESC to quit
* Press 'w' to toggle wrapping of long lines

Moving around
-------------
* Arrow keys
* 'h', 'l' for left and right (as in vim)
* Left / right can be used to hide / show line numbers
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
func NewPager(r *Reader) *Pager {
	return &Pager{
		reader:            r,
		quit:              false,
		firstLineOneBased: 1,
		ShowLineNumbers:   true,
		DeInit:            true,
	}
}

func (p *Pager) addSearchFooter() {
	_, height := p.screen.Size()

	pos := 0
	for _, token := range "Search: " + p.searchString {
		p.screen.SetCell(pos, height-1, twin.NewCell(token, twin.StyleDefault))
		pos++
	}

	// Add a cursor
	p.screen.SetCell(pos, height-1, twin.NewCell(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))
}

func (p *Pager) setFooter(footer string) {
	width, height := p.screen.Size()

	pos := 0
	var footerStyle twin.Style
	if p.StatusBarStyle == STATUSBAR_STYLE_INVERSE {
		footerStyle = twin.StyleDefault.WithAttr(twin.AttrReverse)
	} else if p.StatusBarStyle == STATUSBAR_STYLE_PLAIN {
		footerStyle = twin.StyleDefault
	} else if p.StatusBarStyle == STATUSBAR_STYLE_BOLD {
		footerStyle = twin.StyleDefault.WithAttr(twin.AttrBold)
	} else {
		panic(fmt.Sprint("Unrecognized footer style: ", footerStyle))
	}
	for _, token := range footer {
		p.screen.SetCell(pos, height-1, twin.NewCell(token, footerStyle))
		pos++
	}

	for ; pos < width; pos++ {
		p.screen.SetCell(pos, height-1, twin.NewCell(' ', footerStyle))
	}
}

// Quit leaves the help screen or quits the pager
func (p *Pager) Quit() {
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

func (p *Pager) scrollToSearchHits() {
	if p.searchPattern == nil {
		// This is not a search
		return
	}

	firstHitLine := p.findFirstHitLineOneBased(p.firstLineOneBased, false)
	if firstHitLine == nil {
		// No match, give up
		return
	}

	if *firstHitLine <= p.getLastVisibleLineOneBased() {
		// Already on-screen, never mind
		return
	}

	p.firstLineOneBased = *firstHitLine
}

func (p *Pager) getLastVisibleLineOneBased() int {
	firstVisibleLineOneBased := p.firstLineOneBased
	_, windowHeight := p.screen.Size()

	// If first line is 1 and window is 2 high, and one line is the status
	// line, the last line will be 1 + 2 - 2 = 1
	return firstVisibleLineOneBased + windowHeight - 2
}

func (p *Pager) findFirstHitLineOneBased(firstLineOneBased int, backwards bool) *int {
	lineNumber := firstLineOneBased
	for {
		line := p.reader.GetLine(lineNumber)
		if line == nil {
			// No match, give up
			return nil
		}

		lineText := line.Plain()
		if p.searchPattern.MatchString(lineText) {
			return &lineNumber
		}

		if backwards {
			lineNumber--
		} else {
			lineNumber++
		}
	}
}

func (p *Pager) scrollToNextSearchHit() {
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
		firstSearchLineOneBased = p.getLastVisibleLineOneBased() + 1

	case _NotFound:
		// Restart searching from the top
		p.mode = _Viewing
		firstSearchLineOneBased = 1

	default:
		panic(fmt.Sprint("Unknown search mode when finding next: ", p.mode))
	}

	firstHitLine := p.findFirstHitLineOneBased(firstSearchLineOneBased, false)
	if firstHitLine == nil {
		p.mode = _NotFound
		return
	}
	p.firstLineOneBased = *firstHitLine
}

func (p *Pager) scrollToPreviousSearchHit() {
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

	firstHitLine := p.findFirstHitLineOneBased(firstSearchLineOneBased, true)
	if firstHitLine == nil {
		p.mode = _NotFound
		return
	}
	p.firstLineOneBased = *firstHitLine
}

func (p *Pager) updateSearchPattern() {
	p.searchPattern = toPattern(p.searchString)

	p.scrollToSearchHits()

	// FIXME: If the user is typing, indicate to user if we didn't find anything
}

// toPattern compiles a search string into a pattern.
//
// If the string contains only lower-case letter the pattern will be case insensitive.
//
// If the string is empty the pattern will be nil.
//
// If the string does not compile into a regexp the pattern will match the string verbatim
func toPattern(compileMe string) *regexp.Regexp {
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

// From: https://stackoverflow.com/a/57005674/473672
func removeLastChar(s string) string {
	r, size := utf8.DecodeLastRuneInString(s)
	if r == utf8.RuneError && (size == 0 || size == 1) {
		size = 0
	}
	return s[:len(s)-size]
}

func (p *Pager) scrollToEnd() {
	p.firstLineOneBased = p.reader.GetLineCount()
}

func (p *Pager) onSearchKey(key twin.KeyCode) {
	switch key {
	case twin.KeyEscape, twin.KeyEnter:
		p.mode = _Viewing

	case twin.KeyBackspace, twin.KeyDelete:
		if len(p.searchString) == 0 {
			return
		}

		p.searchString = removeLastChar(p.searchString)
		p.updateSearchPattern()

	case twin.KeyUp:
		// Clipping is done in _Redraw()
		p.firstLineOneBased--
		p.mode = _Viewing

	case twin.KeyDown:
		// Clipping is done in _Redraw()
		p.firstLineOneBased++
		p.mode = _Viewing

	case twin.KeyPgUp:
		_, height := p.screen.Size()
		p.firstLineOneBased -= (height - 1)
		p.mode = _Viewing

	case twin.KeyPgDown:
		_, height := p.screen.Size()
		p.firstLineOneBased += (height - 1)
		p.mode = _Viewing

	default:
		log.Debugf("Unhandled search key event %v", key)
	}
}

func (p *Pager) moveRight(delta int) {
	if p.ShowLineNumbers && delta > 0 {
		p.ShowLineNumbers = false
		return
	}

	if p.leftColumnZeroBased == 0 && delta < 0 {
		p.ShowLineNumbers = true
		return
	}

	result := p.leftColumnZeroBased + delta
	if result < 0 {
		p.leftColumnZeroBased = 0
	} else {
		p.leftColumnZeroBased = result
	}
}

func (p *Pager) onKey(keyCode twin.KeyCode) {
	if p.mode == _Searching {
		p.onSearchKey(keyCode)
		return
	}
	if p.mode != _Viewing && p.mode != _NotFound {
		panic(fmt.Sprint("Unhandled mode: ", p.mode))
	}

	// Reset the not-found marker on non-search keypresses
	p.mode = _Viewing

	switch keyCode {
	case twin.KeyEscape:
		p.Quit()

	case twin.KeyUp:
		// Clipping is done in _Redraw()
		p.firstLineOneBased--

	case twin.KeyDown, twin.KeyEnter:
		// Clipping is done in _Redraw()
		p.firstLineOneBased++

	case twin.KeyRight:
		p.moveRight(16)

	case twin.KeyLeft:
		p.moveRight(-16)

	case twin.KeyHome:
		p.firstLineOneBased = 1

	case twin.KeyEnd:
		p.firstLineOneBased = p.reader.GetLineCount()

	case twin.KeyPgDown:
		_, height := p.screen.Size()
		p.firstLineOneBased += (height - 1)

	case twin.KeyPgUp:
		_, height := p.screen.Size()
		p.firstLineOneBased -= (height - 1)

	default:
		log.Debugf("Unhandled key event %v", keyCode)
	}
}

func (p *Pager) onSearchRune(char rune) {
	p.searchString = p.searchString + string(char)
	p.updateSearchPattern()
}

func (p *Pager) onRune(char rune) {
	if p.mode == _Searching {
		p.onSearchRune(char)
		return
	}
	if p.mode != _Viewing && p.mode != _NotFound {
		panic(fmt.Sprint("Unhandled mode: ", p.mode))
	}

	switch char {
	case 'q':
		p.Quit()

	case '?':
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
		// Clipping is done in _Redraw()
		p.firstLineOneBased--

	case 'j', 'e':
		// Clipping is done in _Redraw()
		p.firstLineOneBased++

	case 'l':
		// vim right
		p.moveRight(16)

	case 'h':
		// vim left
		p.moveRight(-16)

	case '<', 'g':
		p.firstLineOneBased = 1

	case '>', 'G':
		p.scrollToEnd()

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
		p.scrollToNextSearchHit()

	case 'p', 'N':
		p.scrollToPreviousSearchHit()

	case 'w':
		p.WrapLongLines = !p.WrapLongLines

	default:
		log.Debugf("Unhandled rune keypress '%s'", string(char))
	}
}

// StartPaging brings up the pager on screen
func (p *Pager) StartPaging(screen twin.Screen) {
	unprintableStyle = p.UnprintableStyle
	SetManPageFormatFromEnv()

	p.screen = screen

	go func() {
		for {
			// Wait for new lines to appear...
			<-p.reader.moreLinesAdded

			// ... and notify the main loop so it can show them:
			screen.Events() <- eventMoreLinesAvailable{}

			// Delay updates a bit so that we don't waste time refreshing
			// the screen too often.
			//
			// Note that the delay is *after* reacting, this way single-line
			// updates are reacted to immediately, and the first output line
			// read will appear on screen without delay.
			time.Sleep(200 * time.Millisecond)
		}
	}()

	go func() {
		// Spin the spinner as long as contents is still loading
		done := false
		spinnerFrames := [...]string{"/.\\", "-o-", "\\O/", "| |"}
		spinnerIndex := 0
		for {
			// Break this loop on the reader.done signal...
			select {
			case <-p.reader.done:
				done = true
			default:
				// This default case makes this an async read
			}

			if done {
				break
			}

			screen.Events() <- eventSpinnerUpdate{spinnerFrames[spinnerIndex]}
			spinnerIndex++
			if spinnerIndex >= len(spinnerFrames) {
				spinnerIndex = 0
			}

			time.Sleep(200 * time.Millisecond)
		}

		// Empty our spinner, loading done!
		screen.Events() <- eventSpinnerUpdate{""}
	}()

	// Main loop
	spinner := ""
	for !p.quit {
		if len(screen.Events()) == 0 {
			// Nothing more to process for now, redraw the screen!
			p.redraw(spinner)
		}

		event := <-screen.Events()
		switch event := event.(type) {
		case twin.EventKeyCode:
			log.Tracef("Handling key event %d...", event.KeyCode())
			p.onKey(event.KeyCode())

		case twin.EventRune:
			log.Tracef("Handling rune event '%c'/0x%04x...", event.Rune(), event.Rune())
			p.onRune(event.Rune())

		case twin.EventMouse:
			log.Tracef("Handling mouse event %d...", event.Buttons())
			switch event.Buttons() {
			case twin.MouseWheelUp:
				// Clipping is done in _Redraw()
				p.firstLineOneBased--

			case twin.MouseWheelDown:
				// Clipping is done in _Redraw()
				p.firstLineOneBased++

			case twin.MouseWheelLeft:
				p.moveRight(-16)

			case twin.MouseWheelRight:
				p.moveRight(16)
			}

		case twin.EventResize:
			// We'll be implicitly redrawn just by taking another lap in the loop

		case eventMoreLinesAvailable:
			// Doing nothing here is fine; screen will be refreshed on the next
			// iteration of the main loop.

		case eventSpinnerUpdate:
			spinner = event.spinner

		default:
			log.Warnf("Unhandled event type: %v", event)
		}
	}

	if p.reader.err != nil {
		log.Warnf("Reader reported an error: %s", p.reader.err.Error())
	}
}

// After the pager has exited and the normal screen has been restored, you can
// call this method to print the pager contents to screen again, faking
// "leaving" pager contents on screen after exit.
func (p *Pager) ReprintAfterExit() error {
	// Figure out how many screen lines are used by pager contents
	_, height := p.screen.Size()
	heightWithoutFooter := height - 1
	lineCount := len(p.reader.GetLines(p.firstLineOneBased, heightWithoutFooter).lines)

	if lineCount > 0 {
		p.screen.ShowNLines(lineCount)
	}
	return nil
}
