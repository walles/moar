package m

import (
	"fmt"
	"regexp"

	"github.com/walles/moar/twin"
)

type ScreenLines struct {
	inputLines          *InputLines
	lineNumber          screenLineNumber
	leftColumnZeroBased int

	width  int // Display width
	height int // Display height

	searchPattern *regexp.Regexp

	showLineNumbers bool
	wrapLongLines   bool
}

type RenderedLine struct {
	inputLineOneBased int

	// If an input line has been wrapped into two, the part on the second line
	// will have a wrapIndex of 1.
	wrapIndex int

	cells []twin.Cell
}

func (sl *ScreenLines) lastInputLineOneBased() int {
	if sl.inputLines.lines == nil {
		panic("nil input lines, cannot make up a last input line number")
	}

	// Offsets figured out through trial-and-error...
	return sl.inputLines.firstLineOneBased + len(sl.inputLines.lines) - 1
}

// Render screen lines into an array of lines consisting of Cells.
//
// The second return value is the same as firstInputLineOneBased, but decreased
// if needed so that the end of the input is visible.
func (sl *ScreenLines) renderScreenLines() ([][]twin.Cell, screenLineNumber) {
	if sl.inputLines.lines == nil {
		// Empty input, empty output
		return [][]twin.Cell{}, screenLineNumber{}
	}

	if sl.firstInputLineOneBased < 1 {
		sl.firstInputLineOneBased = 1
	}

	if sl.firstInputLineOneBased > sl.lastInputLineOneBased() {
		sl.firstInputLineOneBased = sl.lastInputLineOneBased()
	}

	allPossibleLines := sl.renderAllLines()

	// Find which index in allPossibleLines the user wants to see at the top of
	// the screen
	firstVisibleIndex := -1 // Not found
	for index, line := range allPossibleLines {
		if line.inputLineOneBased == sl.firstInputLineOneBased {
			firstVisibleIndex = index
			break
		}
	}
	if firstVisibleIndex == -1 {
		panic(fmt.Errorf("firstInputLineOneBased %d not found in allPossibleLines size %d",
			sl.firstInputLineOneBased, len(allPossibleLines)))
	}

	// Ensure the firstVisibleIndex is on an input line boundary, we don't
	// support by-screen-line positioning yet!
	for allPossibleLines[firstVisibleIndex].wrapIndex != 0 {
		firstVisibleIndex--
	}

	lastVisibleIndex := firstVisibleIndex + sl.height - 1
	if lastVisibleIndex < len(allPossibleLines) {
		// Screen has enough room for everything, return everything
		return sl.toScreenLinesArray(allPossibleLines, firstVisibleIndex)
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

	// FIXME: Construct the screen lines to return
	return sl.toScreenLinesArray(allPossibleLines, firstVisibleIndex)
}

func (sl *ScreenLines) toScreenLinesArray(allPossibleLines []RenderedLine, firstVisibleIndex int) ([][]twin.Cell, int) {
	firstInputLineOneBased := allPossibleLines[firstVisibleIndex].inputLineOneBased

	screenLines := make([][]twin.Cell, 0, sl.height)
	for index := firstVisibleIndex; ; index++ {
		if len(screenLines) >= sl.height {
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

func (sl *ScreenLines) renderAllLines() []RenderedLine {
	// Count the length of the last line number
	numberPrefixLength := len(formatNumber(uint(sl.lastInputLineOneBased()))) + 1
	if numberPrefixLength < 4 {
		// 4 = space for 3 digits followed by one whitespace
		//
		// https://github.com/walles/moar/issues/38
		numberPrefixLength = 4
	}

	if !sl.showLineNumbers {
		numberPrefixLength = 0
	}

	allLines := make([]RenderedLine, 0)

	for lineIndex, line := range sl.inputLines.lines {
		lineNumber := sl.inputLines.firstLineOneBased + lineIndex

		highlighted := line.HighlightedTokens(sl.searchPattern)
		var wrapped [][]twin.Cell
		if sl.wrapLongLines {
			wrapped = wrapLine(sl.width-numberPrefixLength, highlighted)
		} else {
			// All on one line
			wrapped = [][]twin.Cell{highlighted}
		}

		for wrapIndex, inputLinePart := range wrapped {
			visibleLineNumber := &lineNumber
			if wrapIndex > 0 {
				visibleLineNumber = nil
			}

			allLines = append(allLines, RenderedLine{
				inputLineOneBased: lineNumber,
				wrapIndex:         wrapIndex,
				cells:             sl.createScreenLine(visibleLineNumber, numberPrefixLength, inputLinePart),
			})
		}
	}

	return allLines
}

func (sl *ScreenLines) createScreenLine(lineNumberToShow *int, numberPrefixLength int, contents []twin.Cell) []twin.Cell {
	newLine := make([]twin.Cell, 0, sl.width)
	newLine = append(newLine, createLineNumberPrefix(lineNumberToShow, numberPrefixLength)...)

	startColumn := sl.leftColumnZeroBased
	if startColumn < len(contents) {
		endColumn := sl.leftColumnZeroBased + (sl.width - numberPrefixLength)
		if endColumn > len(contents) {
			endColumn = len(contents)
		}
		newLine = append(newLine, contents[startColumn:endColumn]...)
	}

	// Add scroll left indicator
	if sl.leftColumnZeroBased > 0 && len(contents) > 0 {
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
	if len(contents)+numberPrefixLength-sl.leftColumnZeroBased > sl.width {
		newLine[sl.width-1] = twin.Cell{
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
