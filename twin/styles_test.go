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
		strings.ReplaceAll(StyleDefault.RenderUpdateFrom(style, ColorCount16), "\x1b", "ESC"),
		"ESC]8;;ESC\\")
}

func TestHyperlinkTransitions(t *testing.T) {
	url1 := "file:///Users/johan/src/riff/src/refiner.rs"
	url2 := "file:///Users/johan/src/riff/src/other.rs"

	// No link -> link
	styleNoLink := StyleDefault
	styleWithLink := StyleDefault.WithHyperlink(&url1)
	output := styleWithLink.RenderUpdateFrom(styleNoLink, ColorCount16)
	assert.Equal(t, strings.ReplaceAll(output, "\x1b", "ESC"), "ESC]8;;"+url1+"ESC\\")

	// Link -> different link
	styleWithLink2 := StyleDefault.WithHyperlink(&url2)
	output = styleWithLink2.RenderUpdateFrom(styleWithLink, ColorCount16)
	assert.Equal(t, strings.ReplaceAll(output, "\x1b", "ESC"), "ESC]8;;"+url2+"ESC\\")

	// Link -> no link
	output = styleNoLink.RenderUpdateFrom(styleWithLink, ColorCount16)
	assert.Equal(t, strings.ReplaceAll(output, "\x1b", "ESC"), "ESC]8;;ESC\\")

	// Link -> same link (no output)
	output = styleWithLink.RenderUpdateFrom(styleWithLink, ColorCount16)
	assert.Equal(t, output, "")
}

func TestBoldNoLinkToBoldLink(t *testing.T) {
	url := "file:///Users/johan/src/riff/src/refiner.rs"
	boldNoLink := StyleDefault.WithAttr(AttrBold)
	boldWithLink := boldNoLink.WithHyperlink(&url)

	output := boldWithLink.RenderUpdateFrom(boldNoLink, ColorCount16)
	assert.Equal(t, strings.ReplaceAll(output, "\x1b", "ESC"), "ESC]8;;"+url+"ESC\\")
}
