package m

import "fmt"

type scrollPositionInternal struct {
	// Line number in the input stream
	lineNumberOneBased int

	// Scroll this many screen lines before rendering. Can be negative.
	deltaScreenLines int
}

type scrollPosition struct {
	internalDontTouch scrollPositionInternal
}

// Create a new position, scrolled towards the end of the file
func (s scrollPosition) previousLine(scrollDistance int) scrollPosition {
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
			lineNumberOneBased: s.internalDontTouch.lineNumberOneBased,
			deltaScreenLines:   s.internalDontTouch.deltaScreenLines - scrollDistance,
		},
	}
}

// Create a new position, scrolled towards the end of the file
func (s scrollPosition) nextLine(scrollDistance int) scrollPosition {
	return scrollPosition{
		internalDontTouch: scrollPositionInternal{
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

// Only to be called from the scrollPosition getters!!
//
// Canonicalize the scroll position vs the given pager.
func (si *scrollPositionInternal) canonicalize(pager *Pager) {
	if si.isCanonical(pager) {
		return
	}
	defer si.setCanonical(pager)

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

// Line number in the input stream
func (s *scrollPosition) lineNumberOneBased(pager *Pager) int {
	s.internalDontTouch.canonicalize(pager)
	return s.internalDontTouch.lineNumberOneBased
}

// Scroll this many screen lines before rendering
//
// Always >= 0.
func (s *scrollPosition) deltaScreenLines(pager *Pager) int {
	s.internalDontTouch.canonicalize(pager)
	return s.internalDontTouch.deltaScreenLines
}
