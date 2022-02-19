package m

type scrollPosition struct {
	// Line number in the input stream, zero based
	lineNumber int

	// Scroll this many screen lines before rendering. Can be negative.
	deltaScreenLines int
}

// Create a new position, scrolled towards the end of the file
func (s scrollPosition) previousLine(scrollDistance int) scrollPosition {
	return scrollPosition{
		lineNumber:       s.lineNumber,
		deltaScreenLines: s.deltaScreenLines - 1,
	}
}

// Create a new position, scrolled towards the end of the file
func (s scrollPosition) nextLine(scrollDistance int) scrollPosition {
	return scrollPosition{
		lineNumber:       s.lineNumber,
		deltaScreenLines: s.deltaScreenLines + 1,
	}
}
