package m

import (
	"math"
	"slices"

	"github.com/walles/moar/m/linemetadata"
)

// FIXME: This reader should filter its input lines based on the search query
// from the pager.

type FilteringReader struct {
	backingReader Reader
}

var greenIndices = []linemetadata.Index{
	linemetadata.IndexFromZeroBased(0),
	linemetadata.IndexFromZeroBased(99),
	linemetadata.IndexFromZeroBased(199),
	linemetadata.IndexFromZeroBased(299),
}

func (f FilteringReader) GetLineCount() int {
	return len(f.GetLines(linemetadata.Index{}, math.MaxInt).lines)
}

func (f FilteringReader) GetLine(index linemetadata.Index) *NumberedLine {
	allLines := f.GetLines(linemetadata.Index{}, math.MaxInt)
	if index.Index() < 0 || index.Index() >= len(allLines.lines) {
		return nil
	}
	return allLines.lines[index.Index()]
}

func (f FilteringReader) GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	lines := make([]*NumberedLine, 0)

	allBaseLines := f.backingReader.GetLines(linemetadata.Index{}, math.MaxInt)
	for i, line := range allBaseLines.lines {
		if slices.Contains(greenIndices, line.index) {
			lines = append(lines, &NumberedLine{
				line:   line.line,
				index:  linemetadata.IndexFromZeroBased(i),
				number: line.number,
			})
		}
	}

	return &InputLines{
		lines:      lines[firstLine.Index():],
		statusText: "Filtered lines",
		firstLine:  firstLine,
	}
}
