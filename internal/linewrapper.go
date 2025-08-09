package internal

import (
	"unicode"

	"github.com/walles/moar/twin"
)

// From: https://www.compart.com/en/unicode/U+00A0
//
//revive:disable-next-line:var-naming
const NO_BREAK_SPACE = '\xa0'

// Given some text and a maximum width in screen cells, find the best point at
// which to wrap the text. Return value is in number of runes.
func getWrapCount(line []twin.StyledRune, maxScreenCellsCount int) int {
	screenCells := 0
	bestCutPoint := maxScreenCellsCount
	inLeadingWhitespace := true
	for cutBeforeThisIndex := 0; cutBeforeThisIndex <= maxScreenCellsCount; cutBeforeThisIndex++ {
		canBreakHere := false

		char := line[cutBeforeThisIndex].Rune
		onBreakableSpace := unicode.IsSpace(char) && char != NO_BREAK_SPACE
		if onBreakableSpace && !inLeadingWhitespace {
			// Break-OK whitespace, cut before this one!
			canBreakHere = true
		}

		if !onBreakableSpace {
			inLeadingWhitespace = false
		}

		// Accept cutting inside "]("" in Markdown links: [home](http://127.0.0.1)
		if cutBeforeThisIndex > 0 {
			previousChar := line[cutBeforeThisIndex-1].Rune
			if previousChar == ']' && char == '(' {
				canBreakHere = true
			}
		}

		// Break after single slashes, this is to enable breaking inside URLs / paths
		if cutBeforeThisIndex > 1 {
			beforeSlash := line[cutBeforeThisIndex-2].Rune
			slash := line[cutBeforeThisIndex-1].Rune
			afterSlash := char
			if beforeSlash != '/' && slash == '/' && afterSlash != '/' {
				canBreakHere = true
			}
		}

		if cutBeforeThisIndex > 0 {
			// Break after a hyphen / dash. That's something people do.
			previousChar := line[cutBeforeThisIndex-1].Rune
			if previousChar == '-' && char != '-' {
				canBreakHere = true
			}
		}

		if canBreakHere {
			bestCutPoint = cutBeforeThisIndex
		}

		screenCells += line[cutBeforeThisIndex].Width()
		if screenCells > maxScreenCellsCount {
			// We went too far
			if bestCutPoint > cutBeforeThisIndex {
				// We have to cut here
				bestCutPoint = cutBeforeThisIndex
			}
			break
		}
	}

	return bestCutPoint
}

// How many screen cells wide will this line be?
func getScreenCellCount(runes []twin.StyledRune) int {
	cellCount := 0
	for _, rune := range runes {
		cellCount += rune.Width()
	}

	return cellCount
}

// Wrap one line of text to a maximum width
func wrapLine(width int, line []twin.StyledRune) [][]twin.StyledRune {
	// Trailing space risks showing up by itself on a line, which would just
	// look weird.
	line = twin.TrimSpaceRight(line)

	screenCellCount := getScreenCellCount(line)
	if screenCellCount == 0 {
		return [][]twin.StyledRune{{}}
	}

	wrapped := make([][]twin.StyledRune, 0, len(line)/width)
	for screenCellCount > width {
		wrapWidth := getWrapCount(line, width)
		firstPart := line[:wrapWidth]
		isOnFirstLine := len(wrapped) == 0
		if !isOnFirstLine {
			// Leading whitespace on wrapped lines would just look like
			// indentation, which would be weird for wrapped text.
			firstPart = twin.TrimSpaceLeft(firstPart)
		}

		wrapped = append(wrapped, twin.TrimSpaceRight(firstPart))

		// These runes still need processing
		remaining := twin.TrimSpaceLeft(line[wrapWidth:])

		// Track how many screen cells are left to handle
		handledCount := len(line) - len(remaining)
		screenCellCount -= getScreenCellCount(line[:handledCount])

		// Prepare for the next iteration
		line = remaining
	}

	isOnFirstLine := len(wrapped) == 0
	if !isOnFirstLine {
		// Leading whitespace on wrapped lines would just look like
		// indentation, which would be weird for wrapped text.
		line = twin.TrimSpaceLeft(line)
	}

	if len(line) > 0 {
		wrapped = append(wrapped, twin.TrimSpaceRight(line))
	}

	return wrapped
}
