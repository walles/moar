package m

import (
	"fmt"

	"github.com/walles/moar/twin"
)

type renderedLine struct {
	inputLineOneBased int

	// If an input line has been wrapped into two, the part on the second line
	// will have a wrapIndex of 1.
	wrapIndex int

	cells []twin.Cell

	// Used for rendering clear-to-end-of-line control sequences:
	// https://en.wikipedia.org/wiki/ANSI_escape_code#EL
	//
	// Ref: https://github.com/walles/moar/issues/106
	trailer twin.Style
}

// Refresh the whole pager display, both contents lines and the status line at
// the bottom
func (p *Pager) redraw(spinner string) {
	p.screen.Clear()

	lastUpdatedScreenLineNumber := -1
	var renderedScreenLines [][]twin.Cell
	renderedScreenLines, statusText := p.renderScreenLines()
	for lineNumber, row := range renderedScreenLines {
		lastUpdatedScreenLineNumber = lineNumber
		for column, cell := range row {
			p.screen.SetCell(column, lastUpdatedScreenLineNumber, cell)
		}
	}

	// Status line code follows

	eofSpinner := spinner
	if eofSpinner == "" {
		// This happens when we're done
		eofSpinner = "---"
	}
	spinnerLine := cellsFromString(_EofMarkerFormat + eofSpinner).Cells
	for column, cell := range spinnerLine {
		p.screen.SetCell(column, lastUpdatedScreenLineNumber+1, cell)
	}

	switch p.mode {
	case _Searching:
		p.addSearchFooter()

	case _NotFound:
		p.setFooter("Not found: " + p.searchString)

	case _GotoLine:
		p.addGotoLineFooter()

	case _Viewing:
		helpText := "Press ESC / q to exit, '/' to search, '?' for help"
		if p.isShowingHelp {
			helpText = "Press ESC / q to exit help, '/' to search"
		}

		if p.ShowStatusBar {
			p.setFooter(statusText + spinner + "  " + helpText)
		}

	default:
		panic(fmt.Sprint("Unsupported pager mode: ", p.mode))
	}

	p.screen.Show()
}

// Render screen lines into an array of lines consisting of Cells.
//
// At most height - 1 lines will be returned, leaving room for one status line.
//
// The lines returned by this method are decorated with horizontal scroll
// markers and line numbers and are ready to be output to the screen.
func (p *Pager) renderScreenLines() (lines [][]twin.Cell, statusText string) {
	renderedLines, statusText := p.renderLines()
	if len(renderedLines) == 0 {
		return
	}

	// Construct the screen lines to return
	screenLines := make([][]twin.Cell, 0, len(renderedLines))
	for _, renderedLine := range renderedLines {
		screenLines = append(screenLines, renderedLine.cells)

		if renderedLine.trailer == twin.StyleDefault {
			continue
		}

		// Fill up with the trailer
		screenWidth, _ := p.screen.Size()
		for len(screenLines[len(screenLines)-1]) < screenWidth {
			screenLines[len(screenLines)-1] =
				append(screenLines[len(screenLines)-1], twin.NewCell(' ', renderedLine.trailer))
		}
	}

	return screenLines, statusText
}

// Render all lines that should go on the screen.
//
// Returns both the lines and a suitable status text.
//
// The returned lines are display ready, meaning that they come with horizontal
// scroll markers and line numbers as necessary.
//
// The maximum number of lines returned by this method is limited by the screen
// height. If the status line is visible, you'll get at most one less than the
// screen height from this method.
func (p *Pager) renderLines() ([]renderedLine, string) {
	_, height := p.screen.Size()
	wantedLineCount := height - 1
	if !p.ShowStatusBar {
		wantedLineCount = height
	}

	inputLines := p.reader.GetLines(p.lineNumberOneBased(), wantedLineCount)
	if inputLines.lines == nil {
		// Empty input, empty output
		return []renderedLine{}, inputLines.statusText
	}

	allLines := make([]renderedLine, 0)
	for lineIndex, line := range inputLines.lines {
		lineNumber := inputLines.firstLineOneBased + lineIndex

		allLines = append(allLines, p.renderLine(line, lineNumber)...)
	}

	// Find which index in allLines the user wants to see at the top of the
	// screen
	firstVisibleIndex := -1 // Not found
	for index, line := range allLines {
		if p.lineNumberOneBased() == 0 {
			// Expected zero lines but got some anyway, grab the first one!
			firstVisibleIndex = index
			break
		}
		if line.inputLineOneBased == p.lineNumberOneBased() && line.wrapIndex == p.deltaScreenLines() {
			firstVisibleIndex = index
			break
		}
	}
	if firstVisibleIndex == -1 {
		panic(fmt.Errorf("scrollPosition %#v not found in allLines size %d",
			p.scrollPosition, len(allLines)))
	}

	// Drop the lines that should go above the screen
	allLines = allLines[firstVisibleIndex:]

	if len(allLines) < wantedLineCount {
		// Screen has enough room for everything, return everything
		return allLines, inputLines.statusText
	}

	return allLines[0:wantedLineCount], inputLines.statusText
}

// Render one input line into one or more screen lines.
//
// The returned line is display ready, meaning that it comes with horizontal
// scroll markers and line number as necessary.
//
// lineNumber and numberPrefixLength are required for knowing how much to
// indent, and to (optionally) render the line number.
func (p *Pager) renderLine(line *Line, lineNumber int) []renderedLine {
	highlighted := line.HighlightedTokens(p.searchPattern)
	var wrapped [][]twin.Cell
	if p.WrapLongLines {
		width, _ := p.screen.Size()
		wrapped = wrapLine(width-p.numberPrefixLength(), highlighted.Cells)
	} else {
		// All on one line
		wrapped = [][]twin.Cell{highlighted.Cells}
	}

	rendered := make([]renderedLine, 0)
	for wrapIndex, inputLinePart := range wrapped {
		visibleLineNumber := &lineNumber
		if wrapIndex > 0 {
			visibleLineNumber = nil
		}

		rendered = append(rendered, renderedLine{
			inputLineOneBased: lineNumber,
			wrapIndex:         wrapIndex,
			cells:             p.decorateLine(visibleLineNumber, inputLinePart),
		})
	}

	return rendered
}

// Take a rendered line and decorate as needed:
// * Line number, or leading whitespace for wrapped lines
// * Scroll left indicator
// * Scroll right indicator
func (p *Pager) decorateLine(lineNumberToShow *int, contents []twin.Cell) []twin.Cell {
	width, _ := p.screen.Size()
	newLine := make([]twin.Cell, 0, width)
	numberPrefixLength := p.numberPrefixLength()
	newLine = append(newLine, createLinePrefix(lineNumberToShow, numberPrefixLength)...)

	startColumn := p.leftColumnZeroBased
	if startColumn < len(contents) {
		endColumn := p.leftColumnZeroBased + (width - numberPrefixLength)
		if endColumn > len(contents) {
			endColumn = len(contents)
		}
		newLine = append(newLine, contents[startColumn:endColumn]...)
	}

	// Add scroll left indicator
	if p.leftColumnZeroBased > 0 && len(contents) > 0 {
		if len(newLine) == 0 {
			// Don't panic on short lines, this new Cell will be
			// overwritten with '<' right after this if statement
			newLine = append(newLine, twin.Cell{})
		}

		// Add can-scroll-left marker
		newLine[0] = p.ScrollLeftHint
	}

	// Add scroll right indicator
	if len(contents)+numberPrefixLength-p.leftColumnZeroBased > width {
		newLine[width-1] = p.ScrollRightHint
	}

	return newLine
}

// Generate a line number prefix of the given length.
//
// Can be empty or all-whitespace depending on parameters.
func createLinePrefix(fileLineNumber *int, numberPrefixLength int) []twin.Cell {
	if numberPrefixLength == 0 {
		return []twin.Cell{}
	}

	lineNumberPrefix := make([]twin.Cell, 0, numberPrefixLength)
	if fileLineNumber == nil {
		for len(lineNumberPrefix) < numberPrefixLength {
			lineNumberPrefix = append(lineNumberPrefix, twin.Cell{Rune: ' '})
		}
		return lineNumberPrefix
	}

	lineNumberString := formatNumber(uint(*fileLineNumber))
	lineNumberString = fmt.Sprintf("%*s ", numberPrefixLength-1, lineNumberString)
	if len(lineNumberString) > numberPrefixLength {
		panic(fmt.Errorf(
			"lineNumberString <%s> longer than numberPrefixLength %d",
			lineNumberString, numberPrefixLength))
	}

	for column, digit := range lineNumberString {
		if column >= numberPrefixLength {
			break
		}

		lineNumberPrefix = append(lineNumberPrefix, twin.NewCell(digit, _numberStyle))
	}

	return lineNumberPrefix
}

// Is the given position visible on screen?
func (p *Pager) isVisible(scrollPosition scrollPosition) bool {
	if scrollPosition.lineNumberOneBased(p) < p.lineNumberOneBased() {
		// It's above the screen, not visible
		return false
	}

	lastVisiblePosition := p.getLastVisiblePosition()
	if scrollPosition.lineNumberOneBased(p) > lastVisiblePosition.lineNumberOneBased(p) {
		// Line number too high, not visible
		return false
	}

	if scrollPosition.deltaScreenLines(p) > lastVisiblePosition.deltaScreenLines(p) {
		// Sub-line-number too high, not visible
		return false
	}

	return true
}
