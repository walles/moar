package m

import (
	"fmt"
	"math"
	"regexp"
	"runtime/debug"
	"time"

	"github.com/alecthomas/chroma/v2"
	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/m/linemetadata"
	"github.com/walles/moar/m/reader"
	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
)

type PagerMode interface {
	onKey(key twin.KeyCode)
	onRune(char rune)
	drawFooter(statusText string, spinner string)
}

type StatusBarOption int

const (
	//revive:disable-next-line:var-naming
	STATUSBAR_STYLE_INVERSE StatusBarOption = iota
	//revive:disable-next-line:var-naming
	STATUSBAR_STYLE_PLAIN
	//revive:disable-next-line:var-naming
	STATUSBAR_STYLE_BOLD
)

type eventSpinnerUpdate struct {
	spinner string
}

type eventMoreLinesAvailable struct{}

// Either reading, highlighting or both are done. Check reader.Done() and
// reader.HighlightingDone() for details.
type eventMaybeDone struct{}

// Pager is the main on-screen pager
type Pager struct {
	reader              *reader.ReaderImpl
	filteringReader     FilteringReader
	screen              twin.Screen
	quit                bool
	scrollPosition      scrollPosition
	leftColumnZeroBased int

	// Maybe this should be renamed to "controller"? Because it controls the UI?
	// But since we replace it in a lot of places based on the UI mode, maybe
	// mode is better?
	mode PagerMode

	searchString  string
	searchPattern *regexp.Regexp
	filterPattern *regexp.Regexp

	// We used to have a "Following" field here. If you want to follow, set
	// TargetLineNumber to LineNumberMax() instead, see below.

	isShowingHelp bool
	preHelpState  *_PreHelpState

	// NewPager shows lines by default, this field can hide them
	ShowLineNumbers bool

	StatusBarStyle StatusBarOption
	ShowStatusBar  bool

	UnprintableStyle textstyles.UnprintableStyleT

	WrapLongLines bool

	// Ref: https://github.com/walles/moar/issues/113
	QuitIfOneScreen bool

	// Ref: https://github.com/walles/moar/issues/94
	ScrollLeftHint  twin.StyledRune
	ScrollRightHint twin.StyledRune

	SideScrollAmount int // Should be positive

	// If non-nil, scroll to this line as soon as possible. Set this value to
	// IndexMax() to follow the end of the input (tail).
	//
	// NOTE: Always use setTargetLine() to keep the reader in sync with the
	// pager!
	TargetLine *linemetadata.Index

	// If true, pager will clear the screen on return. If false, pager will
	// clear the last line, and show the cursor.
	DeInit bool

	// If DeInit is false, leave this number of lines for the shell prompt after
	// exiting
	DeInitFalseMargin int

	WithTerminalFg bool // If true, don't set linePrefix

	// Length of the longest line displayed. This is used for limiting scrolling to the right.
	longestLineLength int

	// Bookmarks that you can come back to.
	//
	// Ref: https://github.com/walles/moar/issues/175
	marks map[rune]scrollPosition

	ShouldStartWithSearch bool

	AfterExit func() error
}

type _PreHelpState struct {
	scrollPosition      scrollPosition
	leftColumnZeroBased int
	targetLine          *linemetadata.Index
}

var _HelpReader = reader.NewFromText("Help", `
Welcome to Moar, the nice pager!

Miscellaneous
-------------
* Press 'q' or 'ESC' to quit
* Press 'w' to toggle wrapping of long lines
* Press '=' to toggle showing the status bar at the bottom
* Press 'v' to edit the file in your favorite editor

Moving around
-------------
* Arrow keys
* Alt key plus left / right arrow steps one column at a time
* Left / right can be used to hide / show line numbers
* Home and End for start / end of the document
* 'g' for going to a specific line number
* 'm' sets a mark, you will be asked for a letter to label it with
* ' (single quote) jumps to the mark
* CTRL-p moves to the previous line
* CTRL-n moves to the next line
* PageUp / 'b' and PageDown / 'f'
* SPACE moves down a page
* < / 'gg' to go to the start of the document
* > / 'G' to go to the end of the document
* Half page 'u'p / 'd'own, or CTRL-u / CTRL-d
* RETURN moves down one line

Filtering
---------
Type '&' to start filtering, then type your filter expression.

While filtering, arrow keys, PageUp, PageDown, Home and End work as usual.

Press 'ESC' or RETURN to exit filtering mode.

Searching
---------
* Type / to start searching, then type what you want to find
* Type ? to search backwards, then type what you want to find
* Type RETURN to stop searching, or ESC to skip back to where the search started
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

// NewPager creates a new Pager with default settings
func NewPager(r *reader.ReaderImpl) *Pager {
	var name string
	if r == nil || r.Name == nil || len(*r.Name) == 0 {
		name = "Pager"
	} else {
		name = "Pager " + *r.Name
	}

	pager := Pager{
		reader:           r,
		quit:             false,
		ShowLineNumbers:  true,
		ShowStatusBar:    true,
		DeInit:           true,
		SideScrollAmount: 16,
		ScrollLeftHint:   twin.NewStyledRune('<', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		ScrollRightHint:  twin.NewStyledRune('>', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		scrollPosition:   newScrollPosition(name),
	}

	pager.mode = PagerModeViewing{pager: &pager}
	pager.filteringReader = FilteringReader{
		BackingReader: r,
		FilterPattern: &pager.filterPattern,
	}

	return &pager
}

// How many lines are visible on screen? Depends on screen height and whether or
// not the status bar is visible.
func (p *Pager) visibleHeight() int {
	_, height := p.screen.Size()
	if p.ShowStatusBar {
		return height - 1
	}
	return height
}

// How many cells are needed for this line number?
//
// Returns 0 if line numbers are disabled.
func (p *Pager) getLineNumberPrefixLength(lineNumber linemetadata.Number) int {
	if !p.ShowLineNumbers {
		return 0
	}

	length := len(lineNumber.Format()) + 1 // +1 for the space after the line number

	if length < 4 {
		// 4 = space for 3 digits followed by one whitespace
		//
		// https://github.com/walles/moar/issues/38
		return 4
	}

	return length
}

// Draw the footer string at the bottom using the status bar style
func (p *Pager) setFooter(footer string) {
	width, height := p.screen.Size()

	pos := 0
	for _, token := range footer {
		pos += p.screen.SetCell(pos, height-1, twin.NewStyledRune(token, statusbarStyle))
	}

	for pos < width {
		pos += p.screen.SetCell(pos, height-1, twin.NewStyledRune(' ', statusbarStyle))
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
	p.scrollPosition = p.preHelpState.scrollPosition
	p.leftColumnZeroBased = p.preHelpState.leftColumnZeroBased
	p.setTargetLine(p.preHelpState.targetLine)
	p.preHelpState = nil
}

// Negative deltas move left instead
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

	// If we try to move past the characters when moving right, stop scrolling to
	// avoid moving infinitely into the void.
	if p.leftColumnZeroBased > p.longestLineLength {
		p.leftColumnZeroBased = p.longestLineLength
	}
}

func (p *Pager) Reader() reader.Reader {
	if p.isShowingHelp {
		return _HelpReader
	}
	return &p.filteringReader
}

func (p *Pager) handleScrolledUp() {
	p.setTargetLine(nil)
}

func (p *Pager) handleScrolledDown() {
	if p.isScrolledToEnd() {
		// Follow output
		reallyHigh := linemetadata.IndexMax()
		p.setTargetLine(&reallyHigh)
	} else {
		p.setTargetLine(nil)
	}
}

// Except for setting TargetLine, this method also syncs with the reader so that
// the reader knows how many lines it needs to fetch.
func (p *Pager) setTargetLine(targetLine *linemetadata.Index) {
	log.Trace("Pager: Setting target line to ", targetLine, "...")
	p.TargetLine = targetLine
	if targetLine == nil {
		// No target, just do your thing
		p.reader.SetPauseAfterLines(reader.DEFAULT_PAUSE_AFTER_LINES)
		return
	}

	// The value 1000 here is supposed to be larger than any possible screen
	// height, to give us some lookahead and to avoid fetching too few lines.
	targetValue := targetLine.Index() + 1000
	if targetValue < targetLine.Index() {
		// Overflow detected, clip to max int
		targetValue = math.MaxInt
	}
	if targetValue < reader.DEFAULT_PAUSE_AFTER_LINES {
		targetValue = reader.DEFAULT_PAUSE_AFTER_LINES
	}
	p.reader.SetPauseAfterLines(targetValue)
}

// StartPaging brings up the pager on screen
func (p *Pager) StartPaging(screen twin.Screen, chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter) {
	log.Info("Pager starting")

	defer func() {
		if p.reader.Err != nil {
			log.Warnf("Reader reported an error: %s", p.reader.Err.Error())
		}
	}()

	textstyles.UnprintableStyle = p.UnprintableStyle
	consumeLessTermcapEnvs(chromaStyle, chromaFormatter)
	styleUI(chromaStyle, chromaFormatter, p.StatusBarStyle, p.WithTerminalFg)

	p.screen = screen
	if p.ShouldStartWithSearch {
		p.mode = PagerModeSearch{pager: p}
	} else {
		p.mode = PagerModeViewing{pager: p}
	}
	p.marks = make(map[rune]scrollPosition)

	// Make sure the reader knows how many lines we want
	p.setTargetLine(p.TargetLine)

	go func() {
		defer func() {
			PanicHandler("StartPaging()/moreLinesAvailable", recover(), debug.Stack())
		}()

		for range p.reader.MoreLinesAdded {
			// Notify the main loop about the new lines so it can show them
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
		defer func() {
			PanicHandler("StartPaging()/spinner", recover(), debug.Stack())
		}()

		// Spin the spinner as long as contents is still loading
		spinnerFrames := [...]string{"/.\\", "-o-", "\\O/", "| |"}
		spinnerIndex := 0
		for !p.reader.Done.Load() {
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

	go func() {
		defer func() {
			PanicHandler("StartPaging()/maybeDone", recover(), debug.Stack())
		}()

		for range p.reader.MaybeDone {
			screen.Events() <- eventMaybeDone{}
		}
	}()

	log.Info("Entering pager main loop...")

	// Main loop
	spinner := ""
	for !p.quit {
		if len(screen.Events()) == 0 {
			// Nothing more to process for now, redraw the screen
			p.redraw(spinner)

			// Ref:
			// https://github.com/gwsw/less/blob/ff8869aa0485f7188d942723c9fb50afb1892e62/command.c#L828-L831
			//
			// Note that we do the slow (atomic) checks only if the fast ones (no locking
			// required) passed
			if p.QuitIfOneScreen && !p.isShowingHelp && p.reader.Done.Load() && p.reader.HighlightingDone.Load() {
				width, height := p.screen.Size()
				if fitsOnOneScreen(p.reader, width, height-p.DeInitFalseMargin) {
					// Ref:
					// https://github.com/walles/moar/issues/113#issuecomment-1368294132
					p.ShowLineNumbers = false // Requires a redraw to take effect, see below
					p.DeInit = false
					p.quit = true

					// Without this the line numbers setting ^ won't take effect
					p.redraw(spinner)

					log.Info("Exiting because of --quit-if-one-screen, we fit on one screen and we're done")

					break
				}
			}
		}

		event := <-screen.Events()
		switch event := event.(type) {
		case twin.EventKeyCode:
			log.Tracef("Handling key event %d...", event.KeyCode())
			p.mode.onKey(event.KeyCode())

		case twin.EventRune:
			log.Tracef("Handling rune event '%c'/0x%04x...", event.Rune(), event.Rune())
			p.mode.onRune(event.Rune())

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
				p.moveRight(-p.SideScrollAmount)

			case twin.MouseWheelRight:
				p.moveRight(p.SideScrollAmount)
			}

		case twin.EventResize:
			// We'll be implicitly redrawn just by taking another lap in the loop

		case twin.EventExit:
			log.Info("Got a Twin exit event, exiting")
			return

		case eventMoreLinesAvailable:
			if p.TargetLine != nil {
				// The user wants to scroll down to a specific line number
				if linemetadata.IndexFromLength(p.Reader().GetLineCount()).IsBefore(*p.TargetLine) {
					// Not there yet, keep scrolling
					p.scrollToEnd()
				} else {
					// We see the target, scroll to it
					p.scrollPosition = NewScrollPositionFromIndex(*p.TargetLine, "goToTargetLine")
					p.setTargetLine(nil)
				}
			}

		case eventMaybeDone:
			// Do nothing. We got this just so that we'll do the QuitIfOneScreen
			// check (above) as soon as highlighting is done.

		case eventSpinnerUpdate:
			spinner = event.spinner

		case twin.EventTerminalBackgroundDetected:
			// Do nothing, we don't care about background color updates

		default:
			log.Warnf("Unhandled event type: %v", event)
		}
	}
}

// The height parameter is the terminal height minus the height of the user's
// shell prompt.
//
// This way nothing gets scrolled off screen after we exit.
func fitsOnOneScreen(reader *reader.ReaderImpl, width int, height int) bool {
	if reader.GetLineCount() > height {
		return false
	}

	lines := reader.GetLines(linemetadata.Index{}, reader.GetLineCount())
	for _, line := range lines.Lines {
		rendered := line.HighlightedTokens(twin.StyleDefault, nil, nil).StyledRunes
		if len(rendered) > width {
			// This line is too long to fit on one screen line, no fit
			return false
		}
	}
	return true
}

// After the pager has exited and the normal screen has been restored, you can
// call this method to print the pager contents to screen again, faking
// "leaving" pager contents on screen after exit.
func (p *Pager) ReprintAfterExit() error {
	// Figure out how many screen lines are used by pager contents
	renderedScreenLines, _ := p.renderScreenLines()
	screenLinesCount := len(renderedScreenLines)

	_, screenHeight := p.screen.Size()
	screenHeightWithoutFooter := screenHeight - p.DeInitFalseMargin
	if screenLinesCount > screenHeightWithoutFooter {
		screenLinesCount = screenHeightWithoutFooter
	}

	if screenLinesCount > 0 {
		p.screen.ShowNLines(screenLinesCount)
		fmt.Println()
	}

	return nil
}
