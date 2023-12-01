package m

import (
	"strings"
	"testing"

	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

func TestCanonicalize1000(t *testing.T) {
	pager := Pager{}
	pager.screen = twin.NewFakeScreen(100, 100)
	pager.reader = NewReaderFromText("test", strings.Repeat("a\n", 2000))
	pager.ShowStatusBar = true
	pager.ShowLineNumbers = true
	pager.scrollPosition = newScrollPosition("TestCanonicalize1000")

	scrollPosition := scrollPosition{
		internalDontTouch: scrollPositionInternal{
			lineNumberOneBased: 1000,
			deltaScreenLines:   0,
			name:               "Fake scroll position",
			canonicalizing:     false,
			canonical: scrollPositionCanonical{
				width:              0,
				height:             0,
				showLineNumbers:    false,
				showStatusBar:      false,
				wrapLongLines:      false,
				lineNumberOneBased: 0,
				deltaScreenLines:   0,
			},
		},
	}

	lineNumberOneBased := scrollPosition.lineNumberOneBased(&pager)

	assert.Equal(t, lineNumberOneBased, 1)
}
