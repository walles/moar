package m

import (
	"sync"

	"github.com/walles/moar/m/linemetadata"
)

// FIXME: This reader should filter its input lines based on the search query
// from the pager.
//
// For starters, let's just make it always return the last line and see what happens.

type FilteringReader struct {
	backingReader Reader
}

var mockSingleLine = &NumberedLine{
	number: linemetadata.NumberFromOneBased(5),

	index: linemetadata.IndexFromZeroBased(0),
	line: &Line{
		raw:  "This is dummy response line from the filtering reader",
		lock: sync.Mutex{},
	},
}

func (f FilteringReader) GetLineCount() int {
	return 1
}

func (f FilteringReader) GetLine(index linemetadata.Index) *NumberedLine {
	if index.Index() != 0 {
		return nil
	}

	return mockSingleLine
}

func (f FilteringReader) GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	if firstLine.Index() != 0 {
		return &InputLines{
			statusText: "Dummy status text, index out of bounds",
		}
	}

	return &InputLines{
		lines:      []*NumberedLine{mockSingleLine},
		statusText: "Dummy status text, returning single line",
		firstLine:  linemetadata.IndexFromZeroBased(0),
	}
}
