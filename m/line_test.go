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
	prefix := "X"

	manPageHeading := ""
	for _, char := range headingText {
		manPageHeading += string(char) + "\b" + string(char)
	}

	line := NewLine(manPageHeading)
	highlighted := line.HighlightedTokens(prefix, nil, nil)

	assert.Equal(t, len(highlighted.Cells), len(headingText)+len(prefix))
	for i, cell := range highlighted.Cells {
		if i < len(prefix) {
			// Ignore the prefix when checking
			continue
		}

		assert.Equal(t, cell.Rune, rune(headingText[i-len(prefix)]))
		assert.Equal(t, cell.Style, textstyles.ManPageHeading)
	}
}
