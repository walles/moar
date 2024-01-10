package m

import (
	"testing"

	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

// Repro for not-found part of https://github.com/walles/moar/issues/182
func TestNotFoundFindPrevious(t *testing.T) {
	reader := NewReaderFromText("TestNotFoundFindPrevious", "apa\nbepa\ncepa\ndepa")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 2)

	// Wait for reader to finish reading
	for {
		if reader.done.Load() {
			break
		}
	}

	// Look for a hit on the second line
	pager.searchPattern = toPattern("bepa")

	// Press 'p' to find the previous hit
	pager.mode = PagerModeNotFound{pager: pager}
	pager.mode.onRune('p')

	// We should now be on the second line saying "bepa"
	assert.Equal(t, pager.scrollPosition.lineNumber(pager).AsOneBased(), 2)
	assert.Assert(t, pager.isViewing())
}
