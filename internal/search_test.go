package internal

import (
	"testing"

	"github.com/walles/moor/v2/internal/reader"
	"github.com/walles/moor/v2/twin"
	"gotest.tools/v3/assert"
)

func modeName(pager *Pager) string {
	switch pager.mode.(type) {
	case PagerModeViewing:
		return "Viewing"
	case PagerModeNotFound:
		return "NotFound"
	case PagerModeSearch:
		return "Search"
	case *PagerModeGotoLine:
		return "GotoLine"
	default:
		panic("Unknown pager mode")
	}
}

// Create a pager with three screen lines reading from a six lines stream
func createThreeLinesPager(t *testing.T) *Pager {
	reader := reader.NewFromTextForTesting("", "a\nb\nc\nd\ne\nf\n")

	screen := twin.NewFakeScreen(20, 3)
	pager := NewPager(reader)

	pager.screen = screen

	assert.Equal(t, "Viewing", modeName(pager), "Initial pager state")

	return pager
}

func TestScrollToNextSearchHit_StartAtBottom(t *testing.T) {
	// Create a pager scrolled to the last line
	pager := createThreeLinesPager(t)
	pager.scrollToEnd()

	// Set the search to something that doesn't exist in this pager
	pager.searchString = "xxx"
	pager.searchPattern = toPattern(pager.searchString)

	// Scroll to the next search hit
	pager.scrollToNextSearchHit()

	assert.Equal(t, "NotFound", modeName(pager))
}

func TestScrollToNextSearchHit_StartAtTop(t *testing.T) {
	// Create a pager scrolled to the first line
	pager := createThreeLinesPager(t)

	// Set the search to something that doesn't exist in this pager
	pager.searchString = "xxx"
	pager.searchPattern = toPattern(pager.searchString)

	// Scroll to the next search hit
	pager.scrollToNextSearchHit()

	assert.Equal(t, "NotFound", modeName(pager))
}

func TestScrollToNextSearchHit_WrapAfterNotFound(t *testing.T) {
	// Create a pager scrolled to the last line
	pager := createThreeLinesPager(t)
	pager.scrollToEnd()

	// Search for "a", it's on the first line (ref createThreeLinesPager())
	pager.searchString = "a"
	pager.searchPattern = toPattern(pager.searchString)

	// Scroll to the next search hit, this should take us into _NotFound
	pager.scrollToNextSearchHit()
	assert.Equal(t, "NotFound", modeName(pager))

	// Scroll to the next search hit, this should wrap the search and take us to
	// the top
	pager.scrollToNextSearchHit()
	assert.Equal(t, "Viewing", modeName(pager))
	assert.Assert(t, pager.lineIndex().IsZero())
}

func TestScrollToNextSearchHit_WrapAfterFound(t *testing.T) {
	// Create a pager scrolled to the last line
	pager := createThreeLinesPager(t)
	pager.scrollToEnd()

	// Search for "f", it's on the last line (ref createThreeLinesPager())
	pager.searchString = "f"
	pager.searchPattern = toPattern(pager.searchString)

	// Scroll to the next search hit, this should take us into _NotFound
	pager.scrollToNextSearchHit()
	assert.Equal(t, "NotFound", modeName(pager))

	// Scroll to the next search hit, this should wrap the search and take us
	// back to the bottom again
	pager.scrollToNextSearchHit()
	assert.Equal(t, "Viewing", modeName(pager))
	assert.Equal(t, 4, pager.lineIndex().Index())
}

// Ref: https://github.com/walles/moor/issues/152
func Test152(t *testing.T) {
	// Show a pager on a five lines terminal
	reader := reader.NewFromTextForTesting("", "a\nab\nabc\nabcd\nabcde\nabcdef\n")
	screen := twin.NewFakeScreen(20, 5)
	pager := NewPager(reader)
	pager.screen = screen
	assert.Equal(t, "Viewing", modeName(pager), "Initial pager state")

	// Search for the first not-visible hit
	pager.searchString = "abcde"
	searchMode := PagerModeSearch{pager: pager}
	pager.mode = searchMode

	// Scroll to the next search hit
	searchMode.updateSearchPattern()

	assert.Equal(t, "Search", modeName(pager))
	assert.Equal(t, 2, pager.lineIndex().Index())
}
