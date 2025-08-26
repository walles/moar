package internal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/walles/moor/v2/internal/linemetadata"
	"github.com/walles/moor/v2/internal/reader"
	"github.com/walles/moor/v2/twin"
	"gotest.tools/v3/assert"
)

const screenHeight = 60

// Repro for: https://github.com/walles/moor/issues/166
func testCanonicalize1000(t *testing.T, withStatusBar bool, currentStartLine linemetadata.Index, lastVisibleLine linemetadata.Index) {
	pager := Pager{}
	pager.screen = twin.NewFakeScreen(100, screenHeight)
	pager.reader = reader.NewFromTextForTesting("test", strings.Repeat("a\n", 2000))
	pager.filteringReader = FilteringReader{
		BackingReader: pager.reader,
		FilterPattern: &pager.filterPattern,
	}
	pager.ShowLineNumbers = true
	pager.ShowStatusBar = withStatusBar
	pager.scrollPosition = scrollPosition{
		internalDontTouch: scrollPositionInternal{
			lineIndex:        &currentStartLine,
			deltaScreenLines: 0,
			name:             "findFirstHit",
			canonicalizing:   false,
		},
	}

	lastVisiblePosition := scrollPosition{
		internalDontTouch: scrollPositionInternal{
			lineIndex:        &lastVisibleLine,
			deltaScreenLines: 0,
			name:             "Last Visible Position",
		},
	}

	assert.Equal(t, *lastVisiblePosition.lineIndex(&pager), lastVisibleLine)
}

func TestCanonicalize1000WithStatusBar(t *testing.T) {
	for startLine := 0; startLine < 1500; startLine++ {
		t.Run(fmt.Sprint("startLine=", startLine), func(t *testing.T) {
			testCanonicalize1000(t, true,
				linemetadata.IndexFromZeroBased(startLine),
				linemetadata.IndexFromZeroBased(startLine+screenHeight-2),
			)
		})
	}
}

func TestCanonicalize1000WithoutStatusBar(t *testing.T) {
	for startLine := 0; startLine < 1500; startLine++ {
		t.Run(fmt.Sprint("startLine=", startLine), func(t *testing.T) {
			testCanonicalize1000(t, true,
				linemetadata.IndexFromZeroBased(startLine),
				linemetadata.IndexFromZeroBased(startLine+screenHeight-1),
			)
		})
	}
}

// Repro for https://github.com/walles/moor/issues/313: Rapid scroll
// (deltaScreenLines > 0) crossing from 3 to 4 digits must not panic due to
// too-short number prefix length.
func TestFastScrollAcross1000DoesNotPanic(t *testing.T) {
	// Create 1492 lines of single-char content
	pager := Pager{}
	pager.screen = twin.NewFakeScreen(80, screenHeight)
	pager.reader = reader.NewFromTextForTesting("test", strings.Repeat("x\n", 1492))
	pager.filteringReader = FilteringReader{
		BackingReader: pager.reader,
		FilterPattern: &pager.filterPattern,
	}
	pager.ShowLineNumbers = true

	// Start more than one screen height before the line numbers get longer...
	start := linemetadata.IndexFromZeroBased(900)
	pager.scrollPosition = scrollPosition{
		internalDontTouch: scrollPositionInternal{
			name:             "TestFastScrollAcross1000DoesNotPanic",
			lineIndex:        &start,
			deltaScreenLines: 200, // ... and jump to after the line numbers get longer
		},
	}

	// Trigger rendering (and canonicalization). If the prefix is miscomputed
	// this would previously panic inside createLinePrefix().
	lines, status := pager.renderScreenLines()
	assert.Assert(t, lines != nil) // sanity
	_ = status                     // not asserted here; we only care about not panicking
}
