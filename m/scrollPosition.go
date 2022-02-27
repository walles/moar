package m

import "fmt"

// Please create using newScrollPosition(name)
type scrollPosition struct {
	internalDontTouch scrollPositionInternal
}

func newScrollPosition(name string) scrollPosition {
	if len(name) == 0 {
		panic("Non-empty name required")
	}
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name: name,
		},
	}
}

type scrollPositionInternal struct {
	// Line number in the input stream
	lineNumberOneBased int

	// Scroll this many screen lines before rendering. Can be negative.
	deltaScreenLines int

	name           string
	canonicalizing bool
	canonical      scrollPositionCanonical
}

// If any of these change, we have to recompute the scrollPositionInternal values
type scrollPositionCanonical struct {
	width           int  // From pager
	height          int  // From pager
	showLineNumbers bool // From pager
	wrapLongLines   bool // From pager

	lineNumberOneBased int // From scrollPositionInternal
	deltaScreenLines   int // From scrollPositionInternal
}

func canonicalFromPager(pager *Pager) scrollPositionCanonical {
	width, height := pager.screen.Size()
	return scrollPositionCanonical{
		width:           width,
		height:          height,
		showLineNumbers: pager.ShowLineNumbers,
		wrapLongLines:   pager.WrapLongLines,

		lineNumberOneBased: pager.scrollPosition.internalDontTouch.lineNumberOneBased,
		deltaScreenLines:   pager.scrollPosition.internalDontTouch.deltaScreenLines,
	}
}

// Create a new position, scrolled towards the end of the file
func (s scrollPosition) PreviousLine(scrollDistance int) scrollPosition {
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:               s.internalDontTouch.name,
			lineNumberOneBased: s.internalDontTouch.lineNumberOneBased,
			deltaScreenLines:   s.internalDontTouch.deltaScreenLines - scrollDistance,
		},
	}
}

// Create a new position, scrolled towards the end of the file
func (s scrollPosition) NextLine(scrollDistance int) scrollPosition {
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:               s.internalDontTouch.name,
			lineNumberOneBased: s.internalDontTouch.lineNumberOneBased,
			deltaScreenLines:   s.internalDontTouch.deltaScreenLines + scrollDistance,
		},
	}
}

// Move towards the top until deltaScreenLines is not negative any more
func (si *scrollPositionInternal) handleNegativeDeltaScreenLines(pager *Pager) {
	for si.lineNumberOneBased > 1 && si.deltaScreenLines < 0 {
		// Render the previous line
		previousLine := pager.reader.GetLine(si.lineNumberOneBased - 1)
		previousSubLines := pager.renderLine(previousLine, 0)

		// Adjust lineNumberOneBased and deltaScreenLines to move up into the
		// previous screen line
		si.lineNumberOneBased--
		si.deltaScreenLines += len(previousSubLines)
	}

	if si.lineNumberOneBased == 1 && si.deltaScreenLines < 0 {
		// Don't go above the top line
		si.deltaScreenLines = 0
	}
}

// Move towards the bottom until deltaScreenLines is within range of the
// rendering of the current line
func (si *scrollPositionInternal) handlePositiveDeltaScreenLines(pager *Pager) {
	for {
		line := pager.reader.GetLine(si.lineNumberOneBased)
		if line == nil {
			// Out of bounds downwards, get the last line...
			si.lineNumberOneBased = pager.reader.GetLineCount()
			line = pager.reader.GetLine(si.lineNumberOneBased)
			if line == nil {
				panic(fmt.Errorf("Last line is nil"))
			}
			subLines := len(pager.renderLine(line, 0))

			// ... and go to the bottom of that.
			si.deltaScreenLines = subLines - 1
			return
		}

		subLines := len(pager.renderLine(line, 0))
		if si.deltaScreenLines < subLines {
			// Sublines are within bounds!
			return
		}

		si.lineNumberOneBased++
		si.deltaScreenLines -= subLines
	}
}

// This method assumes si contains a canonical position
func (si *scrollPositionInternal) emptyBottomLinesCount(pager *Pager) int {
	_, height := pager.screen.Size()
	unclaimedViewportLines := height - 1 // Status line takes up one row

	// Start counting where the current input line begins
	unclaimedViewportLines += si.deltaScreenLines

	for {
		lineNumberOneBased := si.lineNumberOneBased
		line := pager.reader.GetLine(lineNumberOneBased)
		if line == nil {
			// No more lines!
			break
		}

		subLines := len(pager.renderLine(line, 0))
		unclaimedViewportLines -= subLines
		if unclaimedViewportLines <= 0 {
			return 0
		}
	}

	return unclaimedViewportLines
}

// Only to be called from the scrollPosition getters!!
//
// Canonicalize the scroll position vs the given pager.
func (si *scrollPositionInternal) canonicalize(pager *Pager) {
	if si.canonical == canonicalFromPager(pager) {
		return
	}

	if si.canonicalizing {
		panic(fmt.Errorf("Scroll position canonicalize() called recursively for %s", si.name))
	}
	si.canonicalizing = true

	defer func() {
		si.canonical = canonicalFromPager(pager)
		si.canonicalizing = false
	}()

	if pager.reader.GetLineCount() == 0 {
		si.lineNumberOneBased = 0
		si.deltaScreenLines = 0
		return
	}

	si.handleNegativeDeltaScreenLines(pager)
	si.handlePositiveDeltaScreenLines(pager)
	emptyBottomLinesCount := si.emptyBottomLinesCount(pager)
	if emptyBottomLinesCount > 0 {
		// First, adjust deltaScreenLines to get us to the top
		si.deltaScreenLines -= emptyBottomLinesCount

		// Then, actually go up that many lines
		si.handleNegativeDeltaScreenLines(pager)
	}
}

func scrollPositionFromLineNumber(name string, lineNumberOneBased int) *scrollPosition {
	return &scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:               name,
			lineNumberOneBased: lineNumberOneBased,
		},
	}
}

// Line number in the input stream, or 0 if nothing has been read
func (p *Pager) lineNumberOneBased() int {
	p.scrollPosition.internalDontTouch.canonicalize(p)
	return p.scrollPosition.internalDontTouch.lineNumberOneBased
}

// Line number in the input stream, or 0 if nothing has been read
func (sp *scrollPosition) lineNumberOneBased(pager *Pager) int {
	sp.internalDontTouch.canonicalize(pager)
	return sp.internalDontTouch.lineNumberOneBased
}

// Scroll this many screen lines before rendering
//
// Always >= 0.
func (p *Pager) deltaScreenLines() int {
	p.scrollPosition.internalDontTouch.canonicalize(p)
	return p.scrollPosition.internalDontTouch.deltaScreenLines
}

// Scroll this many screen lines before rendering
//
// Always >= 0.
func (sp *scrollPosition) deltaScreenLines(pager *Pager) int {
	sp.internalDontTouch.canonicalize(pager)
	return sp.internalDontTouch.deltaScreenLines
}

func (p *Pager) scrollToEnd() {
	lastInputLine := p.reader.GetLineCount()
	inputLine := p.reader.GetLine(lastInputLine)
	screenLines := p.renderLine(inputLine, 0)

	p.scrollPosition.internalDontTouch.lineNumberOneBased = lastInputLine
	p.scrollPosition.internalDontTouch.deltaScreenLines = len(screenLines) - 1
}

func (p *Pager) getLastVisiblePosition() *scrollPosition {
	renderedLines, _ := p.renderLines()
	if len(renderedLines) == 0 {
		return nil
	}

	lastRenderedLine := renderedLines[len(renderedLines)-1]
	return &scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:               "Last Visible Position",
			lineNumberOneBased: lastRenderedLine.inputLineOneBased,
			deltaScreenLines:   lastRenderedLine.wrapIndex,
		},
	}
}
