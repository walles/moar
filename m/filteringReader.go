package m

import (
	"sync"

	"github.com/walles/moar/m/linemetadata"
)

// FIXME: This reader should filter its input lines based on the search query
// from the pager.

type FilteringReader struct {
	backingReader Reader
}

const mockedLineCount = 333

func (f FilteringReader) GetLineCount() int {
	return mockedLineCount
}

func (f FilteringReader) getMockedLines() []*NumberedLine {
	returnMe := make([]*NumberedLine, mockedLineCount)
	for i := range mockedLineCount {
		returnMe[i] = &NumberedLine{
			line: &Line{
				raw:  "This is a mocked line for testing purposes.",
				lock: sync.Mutex{},
			},
			index:  linemetadata.IndexFromZeroBased(i),
			number: linemetadata.NumberFromZeroBased(i * 100),
		}
	}
	return returnMe
}

func (f FilteringReader) GetLine(index linemetadata.Index) *NumberedLine {
	if index.Index() >= mockedLineCount {
		return nil
	}

	return f.getMockedLines()[index.Index()]
}

func (f FilteringReader) GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	if firstLine.Index() >= mockedLineCount {
		return &InputLines{
			statusText: "Dummy status text, index out of bounds",
		}
	}

	return &InputLines{
		lines:      f.getMockedLines()[firstLine.Index():],
		statusText: "Dummy status text, returning single line",
		firstLine:  firstLine,
	}
}
