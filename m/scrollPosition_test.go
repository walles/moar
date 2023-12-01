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
	pager.screen = twin.NewFakeScreen(100, 60)
	pager.reader = NewReaderFromText("test", strings.Repeat("a\n", 2000))
	pager.ShowLineNumbers = true
	pager.ShowStatusBar = true
	pager.scrollPosition = scrollPosition{
		internalDontTouch: scrollPositionInternal{
			lineNumberOneBased: 941,
			deltaScreenLines:   0,
			name:               "findFirstHit",
			canonicalizing:     false,
		},
	}

	lastVisiblePosition := scrollPosition{
		internalDontTouch: scrollPositionInternal{
			lineNumberOneBased: 999,
			deltaScreenLines:   0,
			name:               "Last Visible Position",
		},
	}

	lineNumberOneBased := lastVisiblePosition.lineNumberOneBased(&pager)

	assert.Equal(t, lineNumberOneBased, 42)
}
