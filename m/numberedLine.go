package m

import (
	"regexp"

	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
)

type NumberedLine struct {
	number linenumbers.LineNumber
	line   *Line
}

func (nl *NumberedLine) Plain() string {
	return nl.line.Plain(&nl.number)
}

func (nl *NumberedLine) HighlightedTokens(plainTextStyle twin.Style, search *regexp.Regexp) textstyles.StyledRunesWithTrailer {
	return nl.line.HighlightedTokens(plainTextStyle, search, &nl.number)
}
