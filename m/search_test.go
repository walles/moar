package m

import (
	"strings"
	"testing"

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

	if pager.mode != _Viewing {
		t.Errorf("Expected initial pager state to be %v but got %v", _Viewing, pager.mode)
	}

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

	if pager.mode != _NotFound {
		t.Errorf("Expected state %v but got %v", _NotFound, pager.mode)
	}
}

func TestScrollToNextSearchHit_StartAtTop(t *testing.T) {
	// Create a pager scrolled to the first line
	pager := createThreeLinesPager(t)

	// Set the search to something that doesn't exist in this pager
	pager.searchString = "xxx"
	pager.searchPattern = toPattern(pager.searchString)

	// Scroll to the next search hit
	pager.scrollToNextSearchHit()

	if pager.mode != _NotFound {
		t.Errorf("Expected state %v but got %v", _NotFound, pager.mode)
	}
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
	if pager.mode != _NotFound {
		t.Errorf("Expected state %v but got %v", _NotFound, pager.mode)
	}

	// Scroll to the next search hit, this should wrap the search and take us to
	// the top
	pager.scrollToNextSearchHit()
	if pager.mode != _Viewing {
		t.Errorf("Expected state %v but got %v", _Viewing, pager.mode)
	}
	if pager.lineNumberOneBased() != 1 {
		t.Errorf("Expected line number %v but got %v", 1, pager.lineNumberOneBased())
	}
}
