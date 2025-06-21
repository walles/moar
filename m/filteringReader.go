package m

import "github.com/walles/moar/m/linemetadata"

// FIXME: This reader should filter its input lines based on the search query
// from the pager.
//
// Initially, let's just make it return zero lines and see what happens.

type FilteringReader struct {
	backingReader Reader
}

func (f FilteringReader) GetLineCount() int {
	return 0
}

func (f FilteringReader) GetLine(index linemetadata.Index) *NumberedLine {
	return nil
}

func (f FilteringReader) GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	return &InputLines{
		lines:      nil,
		firstLine:  firstLine,
		statusText: "FIXME: This is a dummy filter to demonstrate the interface",
	}

}
