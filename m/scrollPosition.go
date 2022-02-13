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
