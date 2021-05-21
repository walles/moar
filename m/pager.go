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

// FIXME: Profile the pager while searching through a large file

type _PagerMode int

const (
	_Viewing   _PagerMode = 0
	_Searching _PagerMode = 1
	_NotFound  _PagerMode = 2
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

Quitting
--------
* Press 'q' or ESC to quit

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

func (p *Pager) _AddLine(fileLineNumber *int, numberPrefixLength int, screenLineNumber int, line *Line) {
	screenWidth, _ := p.screen.Size()

	lineNumberString := ""
	if numberPrefixLength > 0 && fileLineNumber != nil {
		lineNumberString = formatNumber(uint(*fileLineNumber))
		if len(lineNumberString) > numberPrefixLength {
			panic(fmt.Errorf(
				"lineNumberString <%s> longer than numberPrefixLength %d",
				lineNumberString, numberPrefixLength))
		}
	} else {
		numberPrefixLength = 0
	}

	for column, digit := range lineNumberString {
		if column >= numberPrefixLength {
			break
		}

		p.screen.SetCell(column, screenLineNumber, twin.NewCell(digit, _numberStyle))
	}

	tokens := createScreenLine(p.leftColumnZeroBased, screenWidth-numberPrefixLength, line, p.searchPattern)
	for column, token := range tokens {
		p.screen.SetCell(column+numberPrefixLength, screenLineNumber, token)
	}
}

func createScreenLine(
	stringIndexAtColumnZero int,
	screenColumnsCount int,
	line *Line,
	search *regexp.Regexp,
) []twin.Cell {
	var returnMe []twin.Cell
	searchHitDelta := 0
	if stringIndexAtColumnZero > 0 {
		// Indicate that it's possible to scroll left
		returnMe = append(returnMe, twin.Cell{
			Rune:  '<',
			Style: twin.StyleDefault.WithAttr(twin.AttrReverse),
		})
		searchHitDelta = -1
	}

	if stringIndexAtColumnZero >= len(line.Tokens()) {
		// Nothing (more) to display, never mind
		return returnMe
	}

	plain := line.Plain()
	matchRanges := getMatchRanges(&plain, search)
	for _, token := range line.Tokens()[stringIndexAtColumnZero:] {
		if len(returnMe) >= screenColumnsCount {
			// We are trying to add a character to the right of the screen.
			// Indicate that this line continues to the right.
			returnMe[len(returnMe)-1] = twin.Cell{
				Rune:  '>',
				Style: twin.StyleDefault.WithAttr(twin.AttrReverse),
			}
			break
		}

		style := token.Style
		if matchRanges.InRange(len(returnMe) + stringIndexAtColumnZero + searchHitDelta) {
			// Search hits in reverse video
			style = style.WithAttr(twin.AttrReverse)
		}

		returnMe = append(returnMe, twin.Cell{
			Rune:  token.Rune,
			Style: style,
		})
	}

	return returnMe
}

func (p *Pager) _AddSearchFooter() {
	_, height := p.screen.Size()

	pos := 0
	for _, token := range "Search: " + p.searchString {
		p.screen.SetCell(pos, height-1, twin.NewCell(token, twin.StyleDefault))
		pos++
	}

	// Add a cursor
	p.screen.SetCell(pos, height-1, twin.NewCell(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))
}

func (p *Pager) _AddLines(spinner string) {
	_, height := p.screen.Size()
	wantedLineCount := height - 1

	lines := p.reader.GetLines(p.firstLineOneBased, wantedLineCount)

	// If we're asking for past-the-end lines, the Reader will clip for us,
	// and we should adapt to that. Otherwise if you scroll 100 lines past
	// the end, you'll then have to scroll 100 lines up again before the
	// display starts scrolling visibly.
	p.firstLineOneBased = lines.firstLineOneBased

	// Count the length of the last line number
	//
	// Offsets figured out through trial-and-error...
	lastLineOneBased := lines.firstLineOneBased + len(lines.lines) - 1
	numberPrefixLength := len(formatNumber(uint(lastLineOneBased))) + 1
	if numberPrefixLength < 4 {
		// 4 = space for 3 digits followed by one whitespace
		//
		// https://github.com/walles/moar/issues/38
		numberPrefixLength = 4
	}

	if !p.ShowLineNumbers {
		numberPrefixLength = 0
	}

	screenLineNumber := 0
	for i, line := range lines.lines {
		lineNumber := p.firstLineOneBased + i
		p._AddLine(&lineNumber, numberPrefixLength, screenLineNumber, line)
		screenLineNumber++
	}

	eofSpinner := spinner
	if eofSpinner == "" {
		// This happens when we're done
		eofSpinner = "---"
	}
	spinnerLine := NewLine(_EofMarkerFormat + eofSpinner)
	p._AddLine(nil, 0, screenLineNumber, spinnerLine)

	switch p.mode {
	case _Searching:
		p._AddSearchFooter()

	case _NotFound:
		p._SetFooter("Not found: " + p.searchString)

	case _Viewing:
		helpText := "Press ESC / q to exit, '/' to search, '?' for help"
		if p.isShowingHelp {
			helpText = "Press ESC / q to exit help, '/' to search"
		}
		p._SetFooter(lines.statusText + spinner + "  " + helpText)

	default:
		panic(fmt.Sprint("Unsupported pager mode: ", p.mode))
	}
}

func (p *Pager) _SetFooter(footer string) {
	width, height := p.screen.Size()

	pos := 0
	footerStyle := twin.StyleDefault.WithAttr(twin.AttrReverse)
	for _, token := range footer {
		p.screen.SetCell(pos, height-1, twin.NewCell(token, footerStyle))
		pos++
	}

	for ; pos < width; pos++ {
		p.screen.SetCell(pos, height-1, twin.NewCell(' ', footerStyle))
	}
}

func (p *Pager) _Redraw(spinner string) {
	p.screen.Clear()

	p._AddLines(spinner)

	p.screen.Show()
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

func (p *Pager) _ScrollToSearchHits() {
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

func (p *Pager) _GetLastVisibleLineOneBased() int {
	firstVisibleLineOneBased := p.firstLineOneBased
	_, windowHeight := p.screen.Size()

	// If first line is 1 and window is 2 high, and one line is the status
	// line, the last line will be 1 + 2 - 2 = 1
	return firstVisibleLineOneBased + windowHeight - 2
}

func (p *Pager) _FindFirstHitLineOneBased(firstLineOneBased int, backwards bool) *int {
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

func (p *Pager) _ScrollToNextSearchHit() {
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

func (p *Pager) _ScrollToPreviousSearchHit() {
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

func (p *Pager) _UpdateSearchPattern() {
	p.searchPattern = toPattern(p.searchString)

	p._ScrollToSearchHits()

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

func (p *Pager) _OnSearchKey(key twin.KeyCode) {
	switch key {
	case twin.KeyEscape, twin.KeyEnter:
		p.mode = _Viewing

	case twin.KeyBackspace, twin.KeyDelete:
		if len(p.searchString) == 0 {
			return
		}

		p.searchString = removeLastChar(p.searchString)
		p._UpdateSearchPattern()

	case twin.KeyUp:
		// Clipping is done in _AddLines()
		p.firstLineOneBased--
		p.mode = _Viewing

	case twin.KeyDown:
		// Clipping is done in _AddLines()
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

func (p *Pager) _MoveRight(delta int) {
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

func (p *Pager) _OnKey(keyCode twin.KeyCode) {
	if p.mode == _Searching {
		p._OnSearchKey(keyCode)
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
		// Clipping is done in _AddLines()
		p.firstLineOneBased--

	case twin.KeyDown, twin.KeyEnter:
		// Clipping is done in _AddLines()
		p.firstLineOneBased++

	case twin.KeyRight:
		p._MoveRight(16)

	case twin.KeyLeft:
		p._MoveRight(-16)

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

func (p *Pager) _OnSearchRune(char rune) {
	p.searchString = p.searchString + string(char)
	p._UpdateSearchPattern()
}

func (p *Pager) _OnRune(char rune) {
	if p.mode == _Searching {
		p._OnSearchRune(char)
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
		// Clipping is done in _AddLines()
		p.firstLineOneBased--

	case 'j', 'e':
		// Clipping is done in _AddLines()
		p.firstLineOneBased++

	case 'l':
		// vim right
		p._MoveRight(16)

	case 'h':
		// vim left
		p._MoveRight(-16)

	case '<', 'g':
		p.firstLineOneBased = 1

	case '>', 'G':
		p.firstLineOneBased = p.reader.GetLineCount()

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
		log.Debugf("Unhandled rune keypress '%s'", string(char))
	}
}

// StartPaging brings up the pager on screen
func (p *Pager) StartPaging(screen twin.Screen) {
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
			p._Redraw(spinner)
		}

		event := <-screen.Events()
		switch event := event.(type) {
		case twin.EventKeyCode:
			log.Tracef("Handling key event %d...", event.KeyCode())
			p._OnKey(event.KeyCode())

		case twin.EventRune:
			log.Tracef("Handling rune event '%c'/0x%04x...", event.Rune(), event.Rune())
			p._OnRune(event.Rune())

		case twin.EventMouse:
			log.Tracef("Handling mouse event %d...", event.Buttons())
			switch event.Buttons() {
			case twin.MouseWheelUp:
				// Clipping is done in _AddLines()
				p.firstLineOneBased--

			case twin.MouseWheelDown:
				// Clipping is done in _AddLines()
				p.firstLineOneBased++

			case twin.MouseWheelLeft:
				p._MoveRight(-16)

			case twin.MouseWheelRight:
				p._MoveRight(16)
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
