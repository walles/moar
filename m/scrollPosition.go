package m

import (
	"fmt"

	"github.com/walles/moar/m/linemetadata"
)

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
	// Index into the array of visible input lines, or nil if nothing has been
	// read yet or there are no lines.
	lineNumber *linemetadata.Index

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
	showStatusBar   bool // From pager
	wrapLongLines   bool // From pager

	lineNumber       *linemetadata.Index // From scrollPositionInternal
	deltaScreenLines int                 // From scrollPositionInternal
}

func canonicalFromPager(pager *Pager) scrollPositionCanonical {
	width, height := pager.screen.Size()
	return scrollPositionCanonical{
		width:           width,
		height:          height,
		showLineNumbers: pager.ShowLineNumbers,
		showStatusBar:   pager.ShowStatusBar,
		wrapLongLines:   pager.WrapLongLines,

		lineNumber:       pager.scrollPosition.internalDontTouch.lineNumber,
		deltaScreenLines: pager.scrollPosition.internalDontTouch.deltaScreenLines,
	}
}

// Create a new position, scrolled towards the beginning of the file
func (sp scrollPosition) PreviousLine(scrollDistance int) scrollPosition {
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:             sp.internalDontTouch.name,
			lineNumber:       sp.internalDontTouch.lineNumber,
			deltaScreenLines: sp.internalDontTouch.deltaScreenLines - scrollDistance,
		},
	}
}

// Create a new position, scrolled towards the end of the file
func (sp scrollPosition) NextLine(scrollDistance int) scrollPosition {
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:             sp.internalDontTouch.name,
			lineNumber:       sp.internalDontTouch.lineNumber,
			deltaScreenLines: sp.internalDontTouch.deltaScreenLines + scrollDistance,
		},
	}
}

// Create a new position, scrolled to the given line number
//
//revive:disable-next-line:unexported-return
func NewScrollPositionFromLineNumber(lineNumber linemetadata.Index, name string) scrollPosition {
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:             name,
			lineNumber:       &lineNumber,
			deltaScreenLines: 0,
		},
	}
}

// Move towards the top until deltaScreenLines is not negative any more
func (si *scrollPositionInternal) handleNegativeDeltaScreenLines(pager *Pager) {
	for !si.lineNumber.IsZero() && si.deltaScreenLines < 0 {
		// Render the previous line
		previousLineNumber := si.lineNumber.NonWrappingAdd(-1)
		previousLine := pager.reader.GetLine(previousLineNumber)
		previousSubLines := pager.renderLine(previousLine, si.getMaxNumberPrefixLength(pager))

		// Adjust lineNumber and deltaScreenLines to move up into the previous
		// screen line
		si.lineNumber = &previousLineNumber
		si.deltaScreenLines += len(previousSubLines)
	}

	if si.lineNumber.IsZero() && si.deltaScreenLines <= 0 {
		// Can't go any higher
		si.deltaScreenLines = 0
		return
	}
}

// Move towards the bottom until deltaScreenLines is within range of the
// rendering of the current line.
//
// This method will not do any screen-height based clipping, so it could be that
// the position is too far down to display after this returns.
func (si *scrollPositionInternal) handlePositiveDeltaScreenLines(pager *Pager) {
	maxPrefixLength := 0
	allPossibleLines := pager.reader.GetLines(*si.lineNumber, pager.visibleHeight())
	if len(allPossibleLines.lines) > 0 {
		lastPossibleLine := allPossibleLines.lines[len(allPossibleLines.lines)-1]
		lastPossibleLineNumber := lastPossibleLine.number
		maxPrefixLength = pager.getLineNumberPrefixLength(lastPossibleLineNumber)
	}

	for {
		line := pager.reader.GetLine(*si.lineNumber)
		if line == nil {
			// Out of bounds downwards, get the last line...
			si.lineNumber = linemetadata.IndexFromLength(pager.reader.GetLineCount())
			line = pager.reader.GetLine(*si.lineNumber)
			if line == nil {
				panic(fmt.Errorf("Last line is nil"))
			}
			subLines := pager.renderLine(line, si.getMaxNumberPrefixLength(pager))

			// ... and go to the bottom of that.
			si.deltaScreenLines = len(subLines) - 1
			return
		}

		subLines := pager.renderLine(line, maxPrefixLength)
		if si.deltaScreenLines < len(subLines) {
			// Sublines are within bounds!
			return
		}

		nextLineNumber := si.lineNumber.NonWrappingAdd(1)
		si.lineNumber = &nextLineNumber
		si.deltaScreenLines -= len(subLines)
	}
}

// This method assumes si contains a canonical position
func (si *scrollPositionInternal) emptyBottomLinesCount(pager *Pager) int {
	unclaimedViewportLines := pager.visibleHeight()
	if pager.reader.GetLineCount() == 0 {
		// No lines available, so all viewport lines are unclaimed
		return unclaimedViewportLines
	}

	// Start counting where the current input line begins
	unclaimedViewportLines += si.deltaScreenLines

	lineNumber := *si.lineNumber

	lastLineNumber := *linemetadata.NumberFromLength(pager.reader.GetLineCount())
	lastLineNumberWidth := pager.getLineNumberPrefixLength(lastLineNumber)

	for {
		line := pager.reader.GetLine(lineNumber)
		if line == nil {
			// No more lines!
			break
		}

		subLines := pager.renderLine(line, lastLineNumberWidth)
		unclaimedViewportLines -= len(subLines)
		if unclaimedViewportLines <= 0 {
			return 0
		}

		// Move to the next line
		lineNumber = lineNumber.NonWrappingAdd(1)
	}

	return unclaimedViewportLines
}

func (si *scrollPositionInternal) isCanonical(pager *Pager) bool {
	if si.canonical.lineNumber == nil {
		// Awaiting initial lines from the reader
		return false
	}

	if si.canonical == canonicalFromPager(pager) {
		return true
	}

	return false
}

// Is the given position visible on screen?
func (sp scrollPosition) isVisible(pager *Pager) bool {
	if sp.internalDontTouch.deltaScreenLines < 0 {
		panic(fmt.Errorf("Negative incoming deltaScreenLines: %#v", sp.internalDontTouch))
	}

	if sp.internalDontTouch.lineNumber.IsBefore(*pager.lineNumber()) {
		// Line number too low, not visible
		return false
	}

	lastVisiblePosition := pager.getLastVisiblePosition()
	if sp.internalDontTouch.lineNumber.IsAfter(*lastVisiblePosition.lineNumber(pager)) {
		// Line number too high, not visible
		return false
	}

	// Line number is within range, now check the sub-line number

	if sp.internalDontTouch.deltaScreenLines > lastVisiblePosition.deltaScreenLines(pager) {
		// Sub-line-number too high, not visible
		return false
	}

	return true
}

// Only to be called from the scrollPosition getters!!
//
// Canonicalize the scroll position vs the given pager. A canonical position can
// just be displayed on screen, it has been clipped both towards the top and
// bottom of the screen, taking into account the screen height.
func (si *scrollPositionInternal) canonicalize(pager *Pager) {
	if si.isCanonical(pager) {
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
		si.lineNumber = nil
		si.deltaScreenLines = 0
		return
	}

	if si.lineNumber == nil {
		// We have lines, but no line number, start at the top
		si.lineNumber = &linemetadata.Index{}
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

func scrollPositionFromLineNumber(name string, lineNumber linemetadata.Index) *scrollPosition {
	return &scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:       name,
			lineNumber: &lineNumber,
		},
	}
}

// Line number in the input stream, or nil if nothing has been read
func (p *Pager) lineNumber() *linemetadata.Index {
	p.scrollPosition.internalDontTouch.canonicalize(p)
	return p.scrollPosition.internalDontTouch.lineNumber
}

// Line number in the input stream, or nil if nothing has been read
func (sp *scrollPosition) lineNumber(pager *Pager) *linemetadata.Index {
	sp.internalDontTouch.canonicalize(pager)
	return sp.internalDontTouch.lineNumber
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
	inputLineCount := p.reader.GetLineCount()
	if inputLineCount == 0 {
		return
	}
	lastInputLineNumber := *linemetadata.IndexFromLength(inputLineCount)

	lastInputLine := p.reader.GetLine(lastInputLineNumber)

	p.scrollPosition.internalDontTouch.lineNumber = &lastInputLineNumber

	// Scroll down enough. We know for sure the last line won't wrap into more
	// lines than the number of characters it contains.
	p.scrollPosition.internalDontTouch.deltaScreenLines = len(lastInputLine.line.raw)

	if p.TargetLineNumber == nil {
		// Start following the end of the file
		//
		// Otherwise, if we're already aiming for some place, don't overwrite
		// that.
		maxLineNumber := linemetadata.IndexMax()
		p.TargetLineNumber = &maxLineNumber
	}
}

// Can be either because Pager.scrollToEnd() was just called or because the user
// has pressed the down arrow enough times.
func (p *Pager) isScrolledToEnd() bool {
	inputLineCount := p.reader.GetLineCount()
	if inputLineCount == 0 {
		// No lines available, which means we can't scroll any further down
		return true
	}
	lastInputLineNumber := *linemetadata.IndexFromLength(inputLineCount)

	visibleLines, _ := p.renderLines()
	lastVisibleLine := visibleLines[len(visibleLines)-1]
	if lastVisibleLine.inputLineNumber != lastInputLineNumber {
		// Last input line is not on the screen
		return false
	}

	// Last line is on screen, now we need to figure out whether we can see all
	// of it
	lastInputLine := p.reader.GetLine(lastInputLineNumber)
	lastInputLineRendered := p.renderLine(lastInputLine, p.getLineNumberPrefixLength(lastInputLineNumber))
	lastRenderedSubLine := lastInputLineRendered[len(lastInputLineRendered)-1]

	// If the last visible subline is the same as the last possible subline then
	// we're at the bottom
	return lastVisibleLine.wrapIndex == lastRenderedSubLine.wrapIndex
}

// Returns nil if there are no lines
func (p *Pager) getLastVisiblePosition() *scrollPosition {
	renderedLines, _ := p.renderLines()
	if len(renderedLines) == 0 {
		return nil
	}

	lastRenderedLine := renderedLines[len(renderedLines)-1]
	return &scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:             "Last Visible Position",
			lineNumber:       &lastRenderedLine.inputLineNumber,
			deltaScreenLines: lastRenderedLine.wrapIndex,
		},
	}
}

func (si *scrollPositionInternal) getMaxNumberPrefixLength(pager *Pager) int {
	maxPossibleIndex := *linemetadata.IndexFromLength(pager.reader.GetLineCount())

	// This is an approximation assuming we don't do any wrapping. Finding the
	// real answer while wrapping requires rendering, which requires the real
	// answer and so on, so we do an approximation here to save us from
	// recursion.
	//
	// Let's improve on demand.
	var index linemetadata.Index
	// Ref: https://github.com/walles/moar/issues/198
	if si.lineNumber != nil {
		index = *si.lineNumber
	}
	maxVisibleIndex := index.NonWrappingAdd(
		si.deltaScreenLines +
			pager.visibleHeight() - 1)
	if maxVisibleIndex.IsAfter(maxPossibleIndex) {
		maxVisibleIndex = maxPossibleIndex
	}

	// Count the length of the last line number
	return pager.getLineNumberPrefixLength(linemetadata.NumberFromZeroBased(maxVisibleIndex.Index()))
}
