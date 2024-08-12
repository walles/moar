package twin

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestHyperlinkToNormal(t *testing.T) {
	url := "http://example.com"

	style := StyleDefault.WithHyperlink(&url)
	assert.Equal(t,
		strings.ReplaceAll(StyleDefault.RenderUpdateFrom(style, ColorCount16), "", "ESC"),
		"ESC]8;;ESC\\")
}
