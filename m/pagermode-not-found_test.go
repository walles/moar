package m

import (
	"testing"

	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

//revive:disable:empty-block

// Repro for not-found part of https://github.com/walles/moar/issues/182
func TestNotFoundFindPrevious(t *testing.T) {
	reader := NewReaderFromText("TestNotFoundFindPrevious", "apa\nbepa\ncepa\ndepa")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 2)

	assert.NilError(t, reader._wait())

	// Look for a hit on the second line
	pager.searchPattern = toPattern("bepa")

	// Press 'p' to find the previous hit
	pager.mode = PagerModeNotFound{pager: pager}
	pager.mode.onRune('p')

	// We should now be on the second line saying "bepa"
	assert.Equal(t, pager.scrollPosition.lineNumber(pager).AsOneBased(), 2)
	assert.Assert(t, pager.isViewing())
}

func TestWrapSearchBackwards(t *testing.T) {
	reader := NewReaderFromText("TestNotFoundFindPrevious", "gold\napa\nbepa\ngold")
	pager := NewPager(reader)
	pager.screen = twin.NewFakeScreen(40, 3)

	assert.NilError(t, reader._wait())

	// Looking for this should take us to the last line
	pager.searchPattern = toPattern("gold")

	// Press 'p' to find the previous hit
	pager.mode = PagerModeNotFound{pager: pager}
	pager.mode.onRune('p')

	// We should now have found gold on the last line. Since the pager is
	// showing two lines on the screen, this puts the pager line number at 3
	// (not 4).
	assert.Equal(t, pager.scrollPosition.lineNumber(pager).AsOneBased(), 3)
	assert.Assert(t, pager.isViewing())
}
