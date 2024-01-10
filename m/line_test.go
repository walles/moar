package m

import (
	"testing"

	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

func TestHighlightedTokensWithManPageHeading(t *testing.T) {
	// Set a marker style we can recognize and test for
	textstyles.ManPageHeading = twin.StyleDefault.WithForeground(twin.NewColor16(2))

	headingText := "JOHAN"

	// For man page headings, this prefix should be ignored
	prefix := "X"

	manPageHeading := ""
	for _, char := range headingText {
		manPageHeading += string(char) + "\b" + string(char)
	}

	line := NewLine(manPageHeading)
	highlighted := line.HighlightedTokens(prefix, nil, nil)

	assert.Equal(t, len(highlighted.Cells), len(headingText))
	for i, cell := range highlighted.Cells {
		assert.Equal(t, cell.Rune, rune(headingText[i]))
		assert.Equal(t, cell.Style, textstyles.ManPageHeading)
	}
}
