package internal

import (
	"fmt"
	"strings"
	"testing"

	"github.com/walles/moar/internal/linemetadata"
	"github.com/walles/moar/internal/reader"
	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

const screenHeight = 60

// Repro for: https://github.com/walles/moar/issues/166
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
