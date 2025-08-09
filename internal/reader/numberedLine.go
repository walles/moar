package reader

import (
	"regexp"

	"github.com/walles/moor/internal/linemetadata"
	"github.com/walles/moor/internal/textstyles"
	"github.com/walles/moor/twin"
)

type NumberedLine struct {
	Index  linemetadata.Index
	Number linemetadata.Number
	Line   *Line
}

func (nl *NumberedLine) Plain() string {
	return nl.Line.Plain(&nl.Index)
}

func (nl *NumberedLine) HighlightedTokens(plainTextStyle twin.Style, standoutStyle *twin.Style, search *regexp.Regexp) textstyles.StyledRunesWithTrailer {
	return nl.Line.HighlightedTokens(plainTextStyle, standoutStyle, search, &nl.Index)
}
