package m

import (
	"regexp"

	"github.com/walles/moar/m/linemetadata"
	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
)

type NumberedLine struct {
	index  linemetadata.Index
	number linemetadata.Number
	line   *Line
}

func (nl *NumberedLine) Plain() string {
	return nl.line.Plain(&nl.index)
}

func (nl *NumberedLine) HighlightedTokens(plainTextStyle twin.Style, search *regexp.Regexp) textstyles.StyledRunesWithTrailer {
	return nl.line.HighlightedTokens(plainTextStyle, search, &nl.index)
}
