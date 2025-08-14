package internal

import (
	"fmt"

	"github.com/walles/moor/v2/internal/linemetadata"
	"github.com/walles/moor/v2/internal/reader"
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
	lineIndex *linemetadata.Index

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

	pagerLineCount int // From pager.Reader().GetLineCount()

	lineIndex        *linemetadata.Index // From scrollPositionInternal
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

		pagerLineCount: pager.Reader().GetLineCount(),

		lineIndex:        pager.scrollPosition.internalDontTouch.lineIndex,
		deltaScreenLines: pager.scrollPosition.internalDontTouch.deltaScreenLines,
	}
}

// Create a new position, scrolled towards the beginning of the file
func (sp scrollPosition) PreviousLine(scrollDistance int) scrollPosition {
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:             sp.internalDontTouch.name,
			lineIndex:        sp.internalDontTouch.lineIndex,
			deltaScreenLines: sp.internalDontTouch.deltaScreenLines - scrollDistance,
		},
	}
}

// Create a new position, scrolled towards the end of the file
func (sp scrollPosition) NextLine(scrollDistance int) scrollPosition {
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:             sp.internalDontTouch.name,
			lineIndex:        sp.internalDontTouch.lineIndex,
			deltaScreenLines: sp.internalDontTouch.deltaScreenLines + scrollDistance,
		},
	}
}

// Create a new position, scrolled to the given line number
//
//revive:disable-next-line:unexported-return
func NewScrollPositionFromIndex(index linemetadata.Index, name string) scrollPosition {
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:             name,
			lineIndex:        &index,
			deltaScreenLines: 0,
		},
	}
}

// Move towards the top until deltaScreenLines is not negative any more
func (si *scrollPositionInternal) handleNegativeDeltaScreenLines(pager *Pager) {
	for !si.lineIndex.IsZero() && si.deltaScreenLines < 0 {
		// Render the previous line
		previousLineIndex := si.lineIndex.NonWrappingAdd(-1)
		previousLine := pager.Reader().GetLine(previousLineIndex)
		previousSubLinesCount := 0
		if previousLine != nil {
			previousSubLines := pager.renderLine(previousLine, si.getMaxNumberPrefixLength(pager))
			previousSubLinesCount = len(previousSubLines)
		}

		// Adjust lineNumber and deltaScreenLines to move up into the previous
		// screen line
		si.lineIndex = &previousLineIndex
		si.deltaScreenLines += previousSubLinesCount
	}

	if si.lineIndex.IsZero() && si.deltaScreenLines <= 0 {
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
	allPossibleLines := pager.Reader().GetLines(*si.lineIndex, pager.visibleHeight())
	if len(allPossibleLines.Lines) > 0 {
		lastPossibleLine := allPossibleLines.Lines[len(allPossibleLines.Lines)-1]
		maxPrefixLength = pager.getLineNumberPrefixLength(lastPossibleLine.Number)
	}

	for {
		line := pager.Reader().GetLine(*si.lineIndex)
		if line == nil {
			// Out of bounds downwards, get the last line...
			si.lineIndex = linemetadata.IndexFromLength(pager.Reader().GetLineCount())
			line = pager.Reader().GetLine(*si.lineIndex)
			if line == nil {
				panic(fmt.Errorf("Last line is nil"))
			}
			subLines := pager.renderLine(line, maxPrefixLength)

			// ... and go to the bottom of that.
			si.deltaScreenLines = len(subLines) - 1
			return
		}

		subLines := pager.renderLine(line, maxPrefixLength)
		if si.deltaScreenLines < len(subLines) {
			// Sublines are within bounds!
			return
		}

		nextLineIndex := si.lineIndex.NonWrappingAdd(1)
		si.lineIndex = &nextLineIndex
		si.deltaScreenLines -= len(subLines)
	}
}

// This method assumes si contains a canonical position
func (si *scrollPositionInternal) emptyBottomLinesCount(pager *Pager) int {
	unclaimedViewportLines := pager.visibleHeight()
	if unclaimedViewportLines == 0 {
		// No lines at all => no lines are empty. Happens (at least) during
		// testing.
		return 0
	}
	if pager.Reader().GetLineCount() == 0 {
		// No lines available, so all viewport lines are unclaimed
		return unclaimedViewportLines
	}

	// Start counting where the current input line begins
	unclaimedViewportLines += si.deltaScreenLines

	lineIndex := *si.lineIndex

	var lastLine reader.NumberedLine
	lastLineIndex := linemetadata.IndexFromZeroBased(lineIndex.Index() + pager.visibleHeight() - 1)
	lastPossibleLineIndex := linemetadata.IndexFromLength(pager.Reader().GetLineCount())
	if lastPossibleLineIndex != nil && lastLineIndex.IsAfter(*lastPossibleLineIndex) {
		lastLineIndex = *lastPossibleLineIndex
	}

	maybeLastLine := pager.Reader().GetLine(lastLineIndex)
	// This check is needed for the unlikely case that we just reformatted
	// the input stream and it just lost some lines.
	if maybeLastLine != nil {
		lastLine = *maybeLastLine
	}
	lastLineNumberWidth := pager.getLineNumberPrefixLength(lastLine.Number)

	for {
		line := pager.Reader().GetLine(lineIndex)
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
		lineIndex = lineIndex.NonWrappingAdd(1)
	}

	return unclaimedViewportLines
}

func (si *scrollPositionInternal) isCanonical(pager *Pager) bool {
	if si.canonical.lineIndex == nil {
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

	if sp.internalDontTouch.lineIndex.IsBefore(*pager.lineIndex()) {
		// Line number too low, not visible
		return false
	}

	lastVisiblePosition := pager.getLastVisiblePosition()
	if sp.internalDontTouch.lineIndex.IsAfter(*lastVisiblePosition.lineIndex(pager)) {
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

	if pager.Reader().GetLineCount() == 0 {
		si.lineIndex = nil
		si.deltaScreenLines = 0
		return
	}

	if si.lineIndex == nil {
		// We have lines, but no line number, start at the top
		si.lineIndex = &linemetadata.Index{}
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

func scrollPositionFromIndex(name string, index linemetadata.Index) *scrollPosition {
	return &scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:      name,
			lineIndex: &index,
		},
	}
}

// Line index in the input stream, or nil if nothing has been read
func (p *Pager) lineIndex() *linemetadata.Index {
	p.scrollPosition.internalDontTouch.canonicalize(p)
	return p.scrollPosition.internalDontTouch.lineIndex
}

// Line index in the input stream, or nil if nothing has been read
func (sp *scrollPosition) lineIndex(pager *Pager) *linemetadata.Index {
	sp.internalDontTouch.canonicalize(pager)
	return sp.internalDontTouch.lineIndex
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
	inputLineCount := p.Reader().GetLineCount()
	if inputLineCount == 0 {
		return
	}
	lastInputIndex := *linemetadata.IndexFromLength(inputLineCount)

	lastInputLine := p.Reader().GetLine(lastInputIndex)

	p.scrollPosition.internalDontTouch.lineIndex = &lastInputIndex

	// Scroll down enough. We know for sure the last line won't wrap into more
	// lines than the number of characters it contains.
	p.scrollPosition.internalDontTouch.deltaScreenLines = len(lastInputLine.Line.Plain(&lastInputLine.Index))

	if p.TargetLine == nil {
		// Start following the end of the file
		//
		// Otherwise, if we're already aiming for some place, don't overwrite
		// that.
		maxLineIndex := linemetadata.IndexMax()
		p.setTargetLine(&maxLineIndex)
	}
}

// Can be either because Pager.scrollToEnd() was just called or because the user
// has pressed the down arrow enough times.
func (p *Pager) isScrolledToEnd() bool {
	inputLineCount := p.Reader().GetLineCount()
	if inputLineCount == 0 {
		// No lines available, which means we can't scroll any further down
		return true
	}
	lastInputLineIndex := *linemetadata.IndexFromLength(inputLineCount)

	visibleLines, _ := p.renderLines()
	lastVisibleLine := visibleLines[len(visibleLines)-1]
	if lastVisibleLine.inputLineIndex != lastInputLineIndex {
		// Last input line is not on the screen
		return false
	}

	// Last line is on screen, now we need to figure out whether we can see all
	// of it
	lastInputLine := p.Reader().GetLine(lastInputLineIndex)
	lastInputLineRendered := p.renderLine(lastInputLine, p.getLineNumberPrefixLength(lastInputLine.Number))
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
			lineIndex:        &lastRenderedLine.inputLineIndex,
			deltaScreenLines: lastRenderedLine.wrapIndex,
		},
	}
}

func (si *scrollPositionInternal) getMaxNumberPrefixLength(pager *Pager) int {
	maxPossibleIndex := *linemetadata.IndexFromLength(pager.Reader().GetLineCount())

	// This is an approximation assuming we don't do any wrapping. Finding the
	// real answer while wrapping requires rendering, which requires the real
	// answer and so on, so we do an approximation here to save us from
	// recursion.
	//
	// Let's improve on demand.
	var index linemetadata.Index
	// Ref: https://github.com/walles/moor/issues/198
	if si.lineIndex != nil {
		index = *si.lineIndex
	}
	maxVisibleIndex := index.NonWrappingAdd(
		si.deltaScreenLines +
			pager.visibleHeight() - 1)
	if maxVisibleIndex.IsAfter(maxPossibleIndex) {
		maxVisibleIndex = maxPossibleIndex
	}

	var number linemetadata.Number
	lastVisibleLine := pager.Reader().GetLine(maxVisibleIndex)

	// nil can happen when the input stream is empty
	if lastVisibleLine != nil {
		number = lastVisibleLine.Number
	}

	// Count the length of the last line number
	return pager.getLineNumberPrefixLength(number)
}
