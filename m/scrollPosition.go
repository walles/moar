package m

type scrollPosition struct {
	// Line number in the input stream, zero based
	lineNumber int

	// If a file line has been broken into two screen lines by wrapping, the
	// first screen line has wrapIndex 0, and the second one 1.
	wrapIndex int

	// Leftmost column displayed, zero based
	leftColumn int
}

func (s scrollPosition) sameOrBefore(otherPosition scrollPosition) bool {
	if s.lineNumber <= otherPosition.lineNumber {
		return true
	}

	if s.lineNumber > otherPosition.lineNumber {
		return false
	}

	// Invariant: Both positions are on the same line

	if s.wrapIndex == otherPosition.wrapIndex {
		return s.leftColumn <= otherPosition.leftColumn
	}

	return s.wrapIndex <= otherPosition.wrapIndex
}

// Create a new position, scrolled one line towards the end of the file
func (s scrollPosition) nextLine() scrollPosition {
	FIXME: Scroll down one line, taking line wrapping into account
}

// Create a new position, scrolled one line towards the end of the file
func (s scrollPosition) previousLine() scrollPosition {
	FIXME: Scroll up one line, taking line wrapping into account
}
