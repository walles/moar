package m

type scrollPosition struct {
	// Line number in the input stream, zero based
	lineNumber int

	// If a file line has been broken into two screen lines by wrapping, the
	// first screen line has wrapIndex 0, and the second one 1.
	wrapIndex int
}

func (s scrollPosition) sameOrBefore(otherPosition scrollPosition) bool {
	if s.lineNumber <= otherPosition.lineNumber {
		return true
	}

	if s.lineNumber > otherPosition.lineNumber {
		return false
	}

	// Invariant: Both positions are on the same line

	return s.wrapIndex <= otherPosition.wrapIndex
}

// Create a new position, scrolled towards the end of the file
func (s scrollPosition) previousLine(pager *Pager, scrollDistance int) scrollPosition {
	// FIXME: Scroll up, taking line wrapping into account
}

// Create a new position, scrolled towards the end of the file
func (s scrollPosition) nextLine(pager *Pager, scrollDistance int) scrollPosition {
	// FIXME: Scroll down, taking line wrapping into account

	returnMe := s

	for i := 0; i < scrollDistance; i++ {
		// FIXME: Split the current line into []RenderedLines
		rendered := renderLine(fixme_rawCurrentLineContents)

		// Move to the next wrapIndex
		returnMe.wrapIndex += 1

		// If our wrapIndex > the maximum RenderedLine wrapIndex, move to the next
		// input line
		if returnMe.wrapIndex > len(rendered) {
			if returnMe.lineNumber == pager.reader.GetLineCount()-1 {
				// We're already at the last line, can't scroll any further
				return returnMe
			}

			returnMe.lineNumber += 1
			returnMe.wrapIndex = 0
		}
	}

	return returnMe
}
