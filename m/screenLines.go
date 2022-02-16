package m

import (
	"fmt"

	"github.com/walles/moar/twin"
)

type RenderedLine struct {
	inputLineOneBased int

	// If an input line has been wrapped into two, the part on the second line
	// will have a wrapIndex of 1.
	wrapIndex int

	cells []twin.Cell
}

// Render screen lines into an array of lines consisting of Cells.
//
// The second return value is the same as firstInputLineOneBased, but decreased
// if needed so that the end of the input is visible.
func (p *Pager) renderScreenLines() (lines [][]twin.Cell, statusText string, newFirstInputLineOneBased int) {
	if p.firstLineOneBased < 1 {
		p.firstLineOneBased = 1
	}

	if p.firstLineOneBased > p._GetLastVisibleLineOneBased() {
		p.firstLineOneBased = p._GetLastVisibleLineOneBased()
	}

	allPossibleLines, statusText := p.renderAllLines()
	if len(allPossibleLines) == 0 {
		return
	}

	// Find which index in allPossibleLines the user wants to see at the top of
	// the screen
	firstVisibleIndex := -1 // Not found
	for index, line := range allPossibleLines {
		if line.inputLineOneBased == p.firstLineOneBased {
			firstVisibleIndex = index
			break
		}
	}
	if firstVisibleIndex == -1 {
		panic(fmt.Errorf("firstInputLineOneBased %d not found in allPossibleLines size %d",
			p.firstLineOneBased, len(allPossibleLines)))
	}

	// Ensure the firstVisibleIndex is on an input line boundary, we don't
	// support by-screen-line positioning yet!
	for allPossibleLines[firstVisibleIndex].wrapIndex != 0 {
		firstVisibleIndex--
	}

	_, height := p.screen.Size()
	lastVisibleIndex := firstVisibleIndex + height - 1
	if lastVisibleIndex < len(allPossibleLines) {
		// Screen has enough room for everything, return everything
		lines, newFirstInputLineOneBased = p.toScreenLinesArray(allPossibleLines, firstVisibleIndex)
		return
	}

	// We seem to be too far down, clip!
	overshoot := 1 + lastVisibleIndex - len(allPossibleLines)
	firstVisibleIndex -= overshoot
	if firstVisibleIndex < 0 {
		firstVisibleIndex = 0
	}

	// Ensure the firstVisibleIndex is on an input line boundary, we don't
	// support by-screen-line positioning yet!
	for firstVisibleIndex > 0 && allPossibleLines[firstVisibleIndex].wrapIndex != 0 {
		firstVisibleIndex--
	}

	// Construct the screen lines to return
	lines, newFirstInputLineOneBased = p.toScreenLinesArray(allPossibleLines, firstVisibleIndex)
	return
}

func (p *Pager) toScreenLinesArray(allPossibleLines []RenderedLine, firstVisibleIndex int) ([][]twin.Cell, int) {
	firstInputLineOneBased := allPossibleLines[firstVisibleIndex].inputLineOneBased

	_, height := p.screen.Size()
	screenLines := make([][]twin.Cell, 0, height)
	for index := firstVisibleIndex; ; index++ {
		if len(screenLines) >= height {
			// All lines rendered, done!
			break
		}
		if index >= len(allPossibleLines) {
			// No more lines available for rendering, done!
			break
		}
		screenLines = append(screenLines, allPossibleLines[index].cells)
	}

	return screenLines, firstInputLineOneBased
}

func (p *Pager) renderAllLines() ([]RenderedLine, string) {
	// Count the length of the last line number
	numberPrefixLength := len(formatNumber(uint(p._GetLastVisibleLineOneBased()))) + 1
	if numberPrefixLength < 4 {
		// 4 = space for 3 digits followed by one whitespace
		//
		// https://github.com/walles/moar/issues/38
		numberPrefixLength = 4
	}

	if !p.ShowLineNumbers {
		numberPrefixLength = 0
	}

	_, height := p.screen.Size()
	wantedLineCount := height - 1
	inputLines := p.reader.GetLines(p.firstLineOneBased, wantedLineCount)
	if inputLines.lines == nil {
		// Empty input, empty output
		return []RenderedLine{}, inputLines.statusText
	}

	allLines := make([]RenderedLine, 0)
	for lineIndex, line := range inputLines.lines {
		lineNumber := inputLines.firstLineOneBased + lineIndex

		allLines = append(allLines, p.renderLine(line, lineNumber, numberPrefixLength)...)
	}

	return allLines, inputLines.statusText
}

// lineNumber and numberPrefixLength are required for knowing how much to
// indent, and to (optionally) render the line number.
func (p *Pager) renderLine(line *Line, lineNumber, numberPrefixLength int) []RenderedLine {
	highlighted := line.HighlightedTokens(p.searchPattern)
	var wrapped [][]twin.Cell
	if p.WrapLongLines {
		width, _ := p.screen.Size()
		wrapped = wrapLine(width-numberPrefixLength, highlighted)
	} else {
		// All on one line
		wrapped = [][]twin.Cell{highlighted}
	}

	rendered := make([]RenderedLine, 0)
	for wrapIndex, inputLinePart := range wrapped {
		visibleLineNumber := &lineNumber
		if wrapIndex > 0 {
			visibleLineNumber = nil
		}

		rendered = append(rendered, RenderedLine{
			inputLineOneBased: lineNumber,
			wrapIndex:         wrapIndex,
			cells:             p.createScreenLine(visibleLineNumber, numberPrefixLength, inputLinePart),
		})
	}

	return rendered
}

func (p *Pager) createScreenLine(lineNumberToShow *int, numberPrefixLength int, contents []twin.Cell) []twin.Cell {
	width, _ := p.screen.Size()
	newLine := make([]twin.Cell, 0, width)
	newLine = append(newLine, createLineNumberPrefix(lineNumberToShow, numberPrefixLength)...)

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
		newLine[0] = twin.Cell{
			Rune:  '<',
			Style: twin.StyleDefault.WithAttr(twin.AttrReverse),
		}
	}

	// Add scroll right indicator
	if len(contents)+numberPrefixLength-p.leftColumnZeroBased > width {
		newLine[width-1] = twin.Cell{
			Rune:  '>',
			Style: twin.StyleDefault.WithAttr(twin.AttrReverse),
		}
	}

	return newLine
}

// Generate a line number prefix. Can be empty or all-whitespace depending on parameters.
func createLineNumberPrefix(fileLineNumber *int, numberPrefixLength int) []twin.Cell {
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
