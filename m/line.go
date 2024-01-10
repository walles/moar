package m

import (
	"regexp"

	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
)

// A Line represents a line of text that can / will be paged
type Line struct {
	raw   string
	plain *string
}

// NewLine creates a new Line from a (potentially ANSI / man page formatted) string
func NewLine(raw string) Line {
	return Line{
		raw:   raw,
		plain: nil,
	}
}

// Returns a representation of the string split into styled tokens. Any regexp
// matches are highlighted. A nil regexp means no highlighting.
func (line *Line) HighlightedTokens(linePrefix string, search *regexp.Regexp, lineNumber *linenumbers.LineNumber) textstyles.CellsWithTrailer {
	plain := line.Plain(lineNumber)
	matchRanges := getMatchRanges(&plain, search)

	fromString := textstyles.CellsFromString(linePrefix, line.raw, lineNumber)
	returnCells := make([]twin.Cell, 0, len(fromString.Cells))
	for _, token := range fromString.Cells {
		style := token.Style
		if matchRanges.InRange(len(returnCells)) {
			if standoutStyle != nil {
				style = *standoutStyle
			} else {
				style = style.WithAttr(twin.AttrReverse)
			}
		}

		returnCells = append(returnCells, twin.Cell{
			Rune:  token.Rune,
			Style: style,
		})
	}

	return textstyles.CellsWithTrailer{
		Cells:   returnCells,
		Trailer: fromString.Trailer,
	}
}

// Plain returns a plain text representation of the initial string
func (line *Line) Plain(lineNumber *linenumbers.LineNumber) string {
	if line.plain == nil {
		plain := textstyles.WithoutFormatting(line.raw, lineNumber)
		line.plain = &plain
	}
	return *line.plain
}
