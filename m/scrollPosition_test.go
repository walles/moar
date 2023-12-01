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
	pager.ShowLineNumbers = true

	// FIXME: Does this matter for reproducing the crash? Recheck after we fix the issue!
	pager.ShowStatusBar = false

	scrollPosition := scrollPosition{
		internalDontTouch: scrollPositionInternal{
			lineNumberOneBased: 1000,
			deltaScreenLines:   0,
			name:               "TestCanonicalize1000",
			canonicalizing:     false,
			canonical:          scrollPositionCanonical{},
		},
	}

	lineNumberOneBased := scrollPosition.lineNumberOneBased(&pager)

	assert.Equal(t, lineNumberOneBased, 1)
}
