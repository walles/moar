package m

import (
	"strings"
	"testing"

	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

// Repro for: https://github.com/walles/moar/issues/166
func TestCanonicalize1000(t *testing.T) {
	pager := Pager{}
	pager.screen = twin.NewFakeScreen(100, 100)
	pager.reader = NewReaderFromText("test", strings.Repeat("a\n", 2000))
	pager.ShowLineNumbers = true

	// FIXME: Does this matter for reproducing the crash? Recheck after we fix the issue!
	pager.ShowStatusBar = true

	pager.scrollPosition = scrollPosition{
		internalDontTouch: scrollPositionInternal{
			lineNumberOneBased: 901,
			deltaScreenLines:   0,
			name:               "TestCanonicalize1000",
			canonicalizing:     false,
			canonical:          scrollPositionCanonical{},
		},
	}
	lineNumberOneBased := pager.scrollPosition.lineNumberOneBased(&pager)

	assert.Equal(t, lineNumberOneBased, 42)
}
