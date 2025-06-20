package m

import (
	"fmt"

	"github.com/walles/moar/m/lines"
	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
)

type renderedLine struct {
	// Which line in the input stream is this?
	inputLineNumber lines.Number

	// If an input line has been wrapped into two, the part on the second line
	// will have a wrapIndex of 1.
	wrapIndex int

	cells []twin.StyledRune

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
	p.longestLineLength = 0

	lastUpdatedScreenLineNumber := -1
	var renderedScreenLines [][]twin.StyledRune
	renderedScreenLines, statusText := p.renderScreenLines()
	for screenLineNumber, row := range renderedScreenLines {
		lastUpdatedScreenLineNumber = screenLineNumber
		column := 0
		for _, cell := range row {
			column += p.screen.SetCell(column, lastUpdatedScreenLineNumber, cell)
		}
	}

	// Status line code follows

	eofSpinner := spinner
	if eofSpinner == "" {
		// This happens when we're done
		eofSpinner = "---"
	}
	spinnerLine := textstyles.StyledRunesFromString(statusbarStyle, eofSpinner, nil).StyledRunes
	column := 0
	for _, cell := range spinnerLine {
		column += p.screen.SetCell(column, lastUpdatedScreenLineNumber+1, cell)
	}

	p.mode.drawFooter(statusText, spinner)

	p.screen.Show()
}

// Render screen lines into an array of lines consisting of Cells.
//
// At most height - 1 lines will be returned, leaving room for one status line.
//
// The lines returned by this method are decorated with horizontal scroll
// markers and line numbers and are ready to be output to the screen.
func (p *Pager) renderScreenLines() (lines [][]twin.StyledRune, statusText string) {
	renderedLines, statusText := p.renderLines()
	if len(renderedLines) == 0 {
		return
	}

	// Construct the screen lines to return
	screenLines := make([][]twin.StyledRune, 0, len(renderedLines))
	for _, renderedLine := range renderedLines {
		screenLines = append(screenLines, renderedLine.cells)

		if renderedLine.trailer == twin.StyleDefault {
			continue
		}

		// Fill up with the trailer
		screenWidth, _ := p.screen.Size()
		for len(screenLines[len(screenLines)-1]) < screenWidth {
			screenLines[len(screenLines)-1] =
				append(screenLines[len(screenLines)-1], twin.NewStyledRune(' ', renderedLine.trailer))
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
	inputLines := p.GetFilteredLines()
	if len(inputLines.lines) == 0 {
		// Empty input, empty output
		return []renderedLine{}, inputLines.statusText
	}

	lastVisibleLineNumber := inputLines.lines[len(inputLines.lines)-1].number
	numberPrefixLength := p.getLineNumberPrefixLength(lastVisibleLineNumber)

	allLines := make([]renderedLine, 0)
	for _, line := range inputLines.lines {
		rendering := p.renderLine(line, numberPrefixLength)

		var onScreenLength int
		for i := range rendering {
			trimmedLen := len(twin.TrimSpaceRight(rendering[i].cells))
			if trimmedLen > onScreenLength {
				onScreenLength = trimmedLen
			}
		}

		// We're trying to find the max length of readable characters to limit
		// the scrolling to right, so we don't go over into the vast emptiness for no reason.
		//
		// The -1 fixed an issue that seemed like an off-by-one where sometimes, when first
		// scrolling completely to the right, the first left scroll did not show the text again.
		displayLength := p.leftColumnZeroBased + onScreenLength - 1

		if displayLength >= p.longestLineLength {
			p.longestLineLength = displayLength
		}

		allLines = append(allLines, rendering...)
	}

	// Find which index in allLines the user wants to see at the top of the
	// screen
	firstVisibleIndex := -1 // Not found
	currentLineNumber := inputLines.firstLine
	for index, line := range allLines {
		if p.lineNumber() == nil {
			// Expected zero lines but got some anyway, grab the first one!
			firstVisibleIndex = index
			break
		}
		if currentLineNumber == *p.lineNumber() && line.wrapIndex == p.deltaScreenLines() {
			firstVisibleIndex = index
			break
		}
		currentLineNumber = currentLineNumber.NonWrappingAdd(1)
	}
	if firstVisibleIndex == -1 {
		panic(fmt.Errorf("scrollPosition %#v not found in allLines size %d",
			p.scrollPosition, len(allLines)))
	}

	// Drop the lines that should go above the screen
	allLines = allLines[firstVisibleIndex:]

	wantedLineCount := p.visibleHeight()
	if len(allLines) <= wantedLineCount {
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
func (p *Pager) renderLine(line *NumberedLine, numberPrefixLength int) []renderedLine {
	highlighted := line.HighlightedTokens(plainTextStyle, p.searchPattern)
	var wrapped [][]twin.StyledRune
	if p.WrapLongLines {
		width, _ := p.screen.Size()
		wrapped = wrapLine(width-numberPrefixLength, highlighted.StyledRunes)
	} else {
		// All on one line
		wrapped = [][]twin.StyledRune{highlighted.StyledRunes}
	}

	rendered := make([]renderedLine, 0)
	for wrapIndex, inputLinePart := range wrapped {
		visibleLineNumber := &line.number
		if wrapIndex > 0 {
			visibleLineNumber = nil
		}

		decorated := p.decorateLine(visibleLineNumber, numberPrefixLength, inputLinePart)

		rendered = append(rendered, renderedLine{
			inputLineNumber: line.number,
			wrapIndex:       wrapIndex,
			cells:           decorated,
		})
	}

	if highlighted.Trailer != twin.StyleDefault {
		// In the presence of wrapping, add the trailer to the last of the wrap
		// lines only. This matches what both iTerm and the macOS Terminal does.
		rendered[len(rendered)-1].trailer = highlighted.Trailer
	}

	return rendered
}

// Take a rendered line and decorate as needed:
//   - Line number, or leading whitespace for wrapped lines
//   - Scroll left indicator
//   - Scroll right indicator
func (p *Pager) decorateLine(lineNumberToShow *lines.Number, numberPrefixLength int, contents []twin.StyledRune) []twin.StyledRune {
	width, _ := p.screen.Size()
	newLine := make([]twin.StyledRune, 0, width)
	newLine = append(newLine, createLinePrefix(lineNumberToShow, numberPrefixLength)...)

	// Find the first and last fully visible runes.
	var firstVisibleRuneIndex *int
	lastVisibleRuneIndex := -1
	screenColumn := numberPrefixLength // Zero based
	lastVisibleScreenColumn := p.leftColumnZeroBased + width - 1
	cutOffRuneToTheLeft := false
	cutOffRuneToTheRight := false
	canScrollRight := false
	for i, char := range contents {
		if firstVisibleRuneIndex == nil && screenColumn >= p.leftColumnZeroBased {
			// Found the first fully visible rune. We need to point to a copy of
			// our loop variable, not the loop variable itself. Just pointing to
			// i, will make firstVisibleRuneIndex point to a new value for every
			// iteration of the loop.
			copyOfI := i
			firstVisibleRuneIndex = &copyOfI
			if i > 0 && screenColumn > p.leftColumnZeroBased && contents[i-1].Width() > 1 {
				// We had to cut a rune in half at the start
				cutOffRuneToTheLeft = true
			}
		}

		screenReached := firstVisibleRuneIndex != nil
		currentCharRightEdge := screenColumn + char.Width() - 1
		beforeRightEdge := currentCharRightEdge <= lastVisibleScreenColumn
		if screenReached {
			if beforeRightEdge {
				// This rune is fully visible
				lastVisibleRuneIndex = i
			} else {
				// We're just outside the screen on the right
				canScrollRight = true

				currentCharLeftEdge := screenColumn
				if currentCharLeftEdge <= lastVisibleScreenColumn {
					// We have to cut this rune in half
					cutOffRuneToTheRight = true
				}

				// Search done, we're off the right edge
				break
			}
		}

		screenColumn += char.Width()
	}

	// Prepend a space if we had to cut a rune in half at the start
	if cutOffRuneToTheLeft {
		newLine = append([]twin.StyledRune{twin.NewStyledRune(' ', p.ScrollLeftHint.Style)}, newLine...)
	}

	// Add the visible runes
	if firstVisibleRuneIndex != nil {
		newLine = append(newLine, contents[*firstVisibleRuneIndex:lastVisibleRuneIndex+1]...)
	}

	// Append a space if we had to cut a rune in half at the end
	if cutOffRuneToTheRight {
		newLine = append(newLine, twin.NewStyledRune(' ', p.ScrollRightHint.Style))
	}

	// Add scroll left indicator
	canScrollLeft := p.leftColumnZeroBased > 0
	if canScrollLeft && len(contents) > 0 {
		if len(newLine) == 0 {
			// Make room for the scroll left indicator
			newLine = make([]twin.StyledRune, 1)
		}

		if newLine[0].Width() > 1 {
			// Replace the first rune with two spaces so we can replace the
			// leftmost cell with a scroll left indicator. First, convert to one
			// space...
			newLine[0] = twin.NewStyledRune(' ', p.ScrollLeftHint.Style)
			// ...then prepend another space:
			newLine = append([]twin.StyledRune{twin.NewStyledRune(' ', p.ScrollLeftHint.Style)}, newLine...)

			// Prepending ref: https://stackoverflow.com/a/53737602/473672
		}

		// Set can-scroll-left marker
		newLine[0] = p.ScrollLeftHint
	}

	// Add scroll right indicator
	if canScrollRight {
		if newLine[len(newLine)-1].Width() > 1 {
			// Replace the last rune with two spaces so we can replace the
			// rightmost cell with a scroll right indicator. First, convert to one
			// space...
			newLine[len(newLine)-1] = twin.NewStyledRune(' ', p.ScrollRightHint.Style)
			// ...then append another space:
			newLine = append(newLine, twin.NewStyledRune(' ', p.ScrollRightHint.Style))
		}

		newLine[len(newLine)-1] = p.ScrollRightHint
	}

	return newLine
}

// Generate a line number prefix of the given length.
//
// Can be empty or all-whitespace depending on parameters.
func createLinePrefix(lineNumber *lines.Number, numberPrefixLength int) []twin.StyledRune {
	if numberPrefixLength == 0 {
		return []twin.StyledRune{}
	}

	lineNumberPrefix := make([]twin.StyledRune, 0, numberPrefixLength)
	if lineNumber == nil {
		for len(lineNumberPrefix) < numberPrefixLength {
			lineNumberPrefix = append(lineNumberPrefix, twin.StyledRune{Rune: ' '})
		}
		return lineNumberPrefix
	}

	lineNumberString := fmt.Sprintf("%*s ", numberPrefixLength-1, lineNumber.Format())
	if len(lineNumberString) > numberPrefixLength {
		panic(fmt.Errorf(
			"lineNumberString <%s> longer than numberPrefixLength %d",
			lineNumberString, numberPrefixLength))
	}

	for column, digit := range lineNumberString {
		if column >= numberPrefixLength {
			break
		}

		lineNumberPrefix = append(lineNumberPrefix, twin.NewStyledRune(digit, lineNumbersStyle))
	}

	return lineNumberPrefix
}
