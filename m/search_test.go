package m

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/walles/moar/twin"
)

// Create a pager with three screen lines reading from a six lines stream
func createThreeLinesPager(t *testing.T) *Pager {
	reader := NewReaderFromStream("", strings.NewReader("a\nb\nc\nd\ne\nf\n"))
	if err := reader._wait(); err != nil {
		panic(err)
	}

	screen := twin.NewFakeScreen(20, 3)
	pager := NewPager(reader)

	pager.screen = screen

	require.Equal(t, _Viewing, pager.mode, "Initial pager state")

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

	require.Equal(t, _NotFound, pager.mode)
}

func TestScrollToNextSearchHit_StartAtTop(t *testing.T) {
	// Create a pager scrolled to the first line
	pager := createThreeLinesPager(t)

	// Set the search to something that doesn't exist in this pager
	pager.searchString = "xxx"
	pager.searchPattern = toPattern(pager.searchString)

	// Scroll to the next search hit
	pager.scrollToNextSearchHit()

	require.Equal(t, _NotFound, pager.mode)
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
	require.Equal(t, _NotFound, pager.mode)

	// Scroll to the next search hit, this should wrap the search and take us to
	// the top
	pager.scrollToNextSearchHit()
	require.Equal(t, _Viewing, pager.mode)
	require.Equal(t, 1, pager.lineNumberOneBased())
}
