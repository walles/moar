package m

import (
	"testing"

	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

// Create a pager with three screen lines reading from a six lines stream
func createThreeLinesPager(t *testing.T) *Pager {
	reader := NewReaderFromText("", "a\nb\nc\nd\ne\nf\n")

	screen := twin.NewFakeScreen(20, 3)
	pager := NewPager(reader)

	pager.screen = screen

	assert.Equal(t, _Viewing, pager.mode, "Initial pager state")

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

	assert.Equal(t, _NotFound, pager.mode)
}

func TestScrollToNextSearchHit_StartAtTop(t *testing.T) {
	// Create a pager scrolled to the first line
	pager := createThreeLinesPager(t)

	// Set the search to something that doesn't exist in this pager
	pager.searchString = "xxx"
	pager.searchPattern = toPattern(pager.searchString)

	// Scroll to the next search hit
	pager.scrollToNextSearchHit()

	assert.Equal(t, _NotFound, pager.mode)
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
	assert.Equal(t, _NotFound, pager.mode)

	// Scroll to the next search hit, this should wrap the search and take us to
	// the top
	pager.scrollToNextSearchHit()
	assert.Equal(t, _Viewing, pager.mode)
	assert.Equal(t, 1, pager.lineNumberOneBased())
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
	assert.Equal(t, _NotFound, pager.mode)

	// Scroll to the next search hit, this should wrap the search and take us
	// back to the bottom again
	pager.scrollToNextSearchHit()
	assert.Equal(t, _Viewing, pager.mode)
	assert.Equal(t, 5, pager.lineNumberOneBased())
}

// Ref: https://github.com/walles/moar/issues/152
func Test152(t *testing.T) {
	// Show a pager on a five lines terminal
	reader := NewReaderFromText("", "a\nab\nabc\nabcd\nabcde\nabcdef\n")
	screen := twin.NewFakeScreen(20, 5)
	pager := NewPager(reader)
	pager.screen = screen
	assert.Equal(t, _Viewing, pager.mode, "Initial pager state")

	// Search for the first not-visible hit
	pager.searchString = "abcde"
	pager.mode = _Searching

	// Scroll to the next search hit
	pager.updateSearchPattern()

	assert.Equal(t, _Searching, pager.mode)
	assert.Equal(t, 3, pager.lineNumberOneBased())
}
