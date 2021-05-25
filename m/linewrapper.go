package m

import (
	"fmt"
	"unicode"

	"github.com/walles/moar/twin"
)

// From: https://www.compart.com/en/unicode/U+00A0
const NO_BREAK_SPACE = '\xa0'

func getWrapWidth(line []twin.Cell, maxWrapWidth int) int {
	if len(line) <= maxWrapWidth {
		panic(fmt.Errorf("cannot compute wrap width when input isn't longer than max (%d<=%d)",
			len(line), maxWrapWidth))
	}

	// Find the last whitespace in the input. Since we want to break *before*
	// whitespace, we loop through characters to the right of the current one.
	for nextIndex := maxWrapWidth; nextIndex > 0; nextIndex-- {
		next := line[nextIndex].Rune
		if unicode.IsSpace(next) && next != NO_BREAK_SPACE {
			// Break-OK whitespace, cut before this one!
			return nextIndex
		}

		current := line[nextIndex-1].Rune
		if current == ']' && next == '(' {
			// Looks like the split in a Markdown link: [text](http://127.0.0.1)
			return nextIndex
		}

		if nextIndex < 2 {
			// Can't check for single slashes
			continue
		}

		// Break after single slashes, this is to enable breaking inside URLs / paths
		previous := line[nextIndex-2].Rune
		if previous != '/' && current == '/' && next != '/' {
			return nextIndex
		}
	}

	// No breakpoint found, give up
	return maxWrapWidth
}

func wrapLine(width int, line []twin.Cell) [][]twin.Cell {
	// Trailing space risks showing up by itself on a line, which would just
	// look weird.
	line = twin.TrimSpaceRight(line)

	if len(line) == 0 {
		return [][]twin.Cell{{}}
	}

	wrapped := make([][]twin.Cell, 0, len(line)/width)
	for len(line) > width {
		wrapWidth := getWrapWidth(line, width)
		firstPart := line[:wrapWidth]
		if len(wrapped) > 0 {
			// Leading whitespace on wrapped lines would just look like
			// indentation, which would be weird for wrapped text.
			firstPart = twin.TrimSpaceLeft(firstPart)
		}

		wrapped = append(wrapped, twin.TrimSpaceRight(firstPart))

		line = twin.TrimSpaceLeft(line[wrapWidth:])
	}

	if len(wrapped) > 0 {
		// Leading whitespace on wrapped lines would just look like
		// indentation, which would be weird for wrapped text.
		line = twin.TrimSpaceLeft(line)
	}

	if len(line) > 0 {
		wrapped = append(wrapped, twin.TrimSpaceRight(line))
	}

	return wrapped
}
