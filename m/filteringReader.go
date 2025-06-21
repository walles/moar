package m

import "github.com/walles/moar/m/linemetadata"

// FIXME: This reader should filter its input lines based on the search query
// from the pager.
//
// For starters, let's just make it return at most one line and see what happens.

type FilteringReader struct {
	backingReader Reader
}

func (f FilteringReader) GetLineCount() int {
	return min(f.backingReader.GetLineCount(), 1)
}

func (f FilteringReader) GetLine(index linemetadata.Index) *NumberedLine {
	if index.Index() > 0 {
		return nil
	}
	return f.backingReader.GetLine(index)
}

func (f FilteringReader) GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	if firstLine.Index() > 0 {
		return &InputLines{
			statusText: "FIXME: This is a dummy filter to demonstrate the interface",
		}
	}

	return f.backingReader.GetLines(firstLine, min(wantedLineCount, 1))
}
