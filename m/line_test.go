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

	manPageHeading := ""
	for _, char := range headingText {
		manPageHeading += string(char) + "\b" + string(char)
	}

	line := NewLine(manPageHeading)
	highlighted := line.HighlightedTokens(twin.StyleDefault, nil, nil)

	assert.Equal(t, len(highlighted.StyledRunes), len(headingText))
	for i, cell := range highlighted.StyledRunes {
		assert.Equal(t, cell.Rune, rune(headingText[i]))
		assert.Equal(t, cell.Style, textstyles.ManPageHeading)
	}
}
