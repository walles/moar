package m

import (
	"fmt"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/twin"
)

type _PagerMode int

const (
	_Viewing _PagerMode = iota
	_Searching
	_NotFound
	_GotoLine
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
	scrollPosition      scrollPosition
	leftColumnZeroBased int

	mode           _PagerMode
	searchString   string
	searchPattern  *regexp.Regexp
	gotoLineString string
	Following      bool

	isShowingHelp bool
	preHelpState  *_PreHelpState

	// NewPager shows lines by default, this field can hide them
	ShowLineNumbers bool

	StatusBarStyle StatusBarStyle
	ShowStatusBar  bool

	UnprintableStyle UnprintableStyle

	WrapLongLines bool

	// Ref: https://github.com/walles/moar/issues/94
	ScrollLeftHint  twin.Cell
	ScrollRightHint twin.Cell

	// If true, pager will clear the screen on return. If false, pager will
	// clear the last line, and show the cursor.
	DeInit bool
}

type _PreHelpState struct {
	reader              *Reader
	scrollPosition      scrollPosition
	leftColumnZeroBased int
	following           bool
}

const _EofMarkerFormat = "\x1b[7m" // Reverse video

var _HelpReader = NewReaderFromText("Help", `
Welcome to Moar, the nice pager!

Miscellaneous
-------------
* Press 'q' or ESC to quit
* Press 'w' to toggle wrapping of long lines
* Press '=' to toggle showing the status bar at the bottom

Moving around
-------------
* Arrow keys
* SPACE moves down a page
* 'g' for going to a specific line number
* Left / right can be used to hide / show line numbers
* PageUp / 'b' and PageDown / 'f'
* Home and End for start / end of the document
* < to go to the start of the document
* > / 'G' to go to the end of the document
* 'h', 'l' for left and right (as in vim)
* Half page 'u'p / 'd'own, or CTRL-u / CTRL-d
* RETURN moves down one line

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
Put the following line in your ~/.bashrc, ~/.bash_profile or ~/.zshrc:
  export PAGER=moar

Source Code
-----------
Available at https://github.com/walles/moar/.
`)

func (pm _PagerMode) isViewing() bool {
	return pm == _Viewing || pm == _NotFound
}

// NewPager creates a new Pager
func NewPager(r *Reader) *Pager {
	var name string
	if r == nil || r.name == nil || len(*r.name) == 0 {
		name = "Pager"
	} else {
		name = "Pager " + *r.name
	}
	return &Pager{
		reader:          r,
		quit:            false,
		ShowLineNumbers: true,
		ShowStatusBar:   true,
		DeInit:          true,
		ScrollLeftHint:  twin.NewCell('<', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		ScrollRightHint: twin.NewCell('>', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		scrollPosition:  newScrollPosition(name),
	}
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
	p.scrollPosition = p.preHelpState.scrollPosition
	p.leftColumnZeroBased = p.preHelpState.leftColumnZeroBased
	p.Following = p.preHelpState.following
	p.preHelpState = nil
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

func (p *Pager) handleScrolledUp() {
	p.Following = false
}

func (p *Pager) handleScrolledDown() {
	p.Following = p.isScrolledToEnd()
}

func (p *Pager) onKey(keyCode twin.KeyCode) {
	if p.mode == _Searching {
		p.onSearchKey(keyCode)
		return
	}
	if p.mode == _GotoLine {
		p.onGotoLineKey(keyCode)
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
		p.scrollPosition = p.scrollPosition.PreviousLine(1)
		p.handleScrolledUp()

	case twin.KeyDown, twin.KeyEnter:
		// Clipping is done in _Redraw()
		p.scrollPosition = p.scrollPosition.NextLine(1)
		p.handleScrolledDown()

	case twin.KeyRight:
		p.moveRight(16)

	case twin.KeyLeft:
		p.moveRight(-16)

	case twin.KeyHome:
		p.scrollPosition = newScrollPosition("Pager scroll position")
		p.handleScrolledUp()

	case twin.KeyEnd:
		p.scrollToEnd()

	case twin.KeyPgUp:
		_, height := p.screen.Size()
		p.scrollPosition = p.scrollPosition.PreviousLine(height - 1)
		p.handleScrolledUp()

	case twin.KeyPgDown:
		_, height := p.screen.Size()
		p.scrollPosition = p.scrollPosition.NextLine(height - 1)
		p.handleScrolledDown()

	default:
		log.Debugf("Unhandled key event %v", keyCode)
	}
}

func (p *Pager) onRune(char rune) {
	if p.mode == _Searching {
		p.onSearchRune(char)
		return
	}
	if p.mode == _GotoLine {
		p.onGotoLineRune(char)
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
				scrollPosition:      p.scrollPosition,
				leftColumnZeroBased: p.leftColumnZeroBased,
				following:           p.Following,
			}
			p.reader = _HelpReader
			p.scrollPosition = newScrollPosition("Pager scroll position")
			p.leftColumnZeroBased = 0
			p.Following = false
			p.isShowingHelp = true
		}

	case '=':
		p.ShowStatusBar = !p.ShowStatusBar

	case 'k', 'y':
		// Clipping is done in _Redraw()
		p.scrollPosition = p.scrollPosition.PreviousLine(1)
		p.handleScrolledUp()

	case 'j', 'e':
		// Clipping is done in _Redraw()
		p.scrollPosition = p.scrollPosition.NextLine(1)
		p.handleScrolledDown()

	case 'l':
		// vim right
		p.moveRight(16)

	case 'h':
		// vim left
		p.moveRight(-16)

	case '<':
		p.scrollPosition = newScrollPosition("Pager scroll position")
		p.handleScrolledUp()

	case '>', 'G':
		p.scrollToEnd()

	case 'f', ' ':
		_, height := p.screen.Size()
		p.scrollPosition = p.scrollPosition.NextLine(height - 1)
		p.handleScrolledDown()

	case 'b':
		_, height := p.screen.Size()
		p.scrollPosition = p.scrollPosition.PreviousLine(height - 1)
		p.handleScrolledUp()

	// '\x15' = CTRL-u, should work like just 'u'.
	// Ref: https://github.com/walles/moar/issues/90
	case 'u', '\x15':
		_, height := p.screen.Size()
		p.scrollPosition = p.scrollPosition.PreviousLine(height / 2)
		p.handleScrolledUp()

	// '\x04' = CTRL-d, should work like just 'd'.
	// Ref: https://github.com/walles/moar/issues/90
	case 'd', '\x04':
		_, height := p.screen.Size()
		p.scrollPosition = p.scrollPosition.NextLine(height / 2)
		p.handleScrolledDown()

	case '/':
		p.mode = _Searching
		p.searchString = ""
		p.searchPattern = nil

	case 'g':
		p.mode = _GotoLine
		p.gotoLineString = ""

	case 'n':
		p.scrollToNextSearchHit()

	case 'p', 'N':
		p.scrollToPreviousSearchHit()

	case 'w':
		p.WrapLongLines = !p.WrapLongLines

	default:
		log.Debugf("Unhandled rune keypress '%s'/0x%08x", string(char), int32(char))
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
				p.scrollPosition = p.scrollPosition.PreviousLine(1)

			case twin.MouseWheelDown:
				// Clipping is done in _Redraw()
				p.scrollPosition = p.scrollPosition.NextLine(1)

			case twin.MouseWheelLeft:
				p.moveRight(-16)

			case twin.MouseWheelRight:
				p.moveRight(16)
			}

		case twin.EventResize:
			// We'll be implicitly redrawn just by taking another lap in the loop

		case eventMoreLinesAvailable:
			if p.mode.isViewing() && p.Following {
				p.scrollToEnd()
			} else {
				// Doing nothing here is fine; screen will be refreshed on the next
				// iteration of the main loop.
			}

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

	renderedScreenLines, _ := p.renderScreenLines()
	screenLinesCount := len(renderedScreenLines)

	_, screenHeight := p.screen.Size()
	screenHeightWithoutFooter := screenHeight - 1
	if screenLinesCount > screenHeightWithoutFooter {
		screenLinesCount = screenHeightWithoutFooter
	}

	if screenLinesCount > 0 {
		p.screen.ShowNLines(screenLinesCount)
	}
	return nil
}
