package m

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
	for si.deltaScreenLines < 0 {
		// FIXME: Render the previous line
		CODE MISSING HERE

		// FIXME: Adjust lineNumberOneBased and deltaScreenLines to move up into
		// the previous screen line
	}
}

// Move towards the bottom until deltaScreenLines is within range of the
// rendering of the current line
func (si *scrollPositionInternal) handlePositiveDeltaScreenLines(pager *Pager) {
	// FIXME: Render the current line
	CODE MISSING HERE

	// FIXME: If deltaScreenLines is outside of the number of screen lines used
	// up by the current input line, adjust lineNumberOneBased and
	// deltaScreenLines to move down and try again
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
	if (there are empty lines at the bottom of the screen) {
		// FIXME: Adjust deltaScreenLines to get us to the top
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
