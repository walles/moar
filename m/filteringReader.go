package m

import (
	"math"
	"regexp"

	"github.com/walles/moar/m/linemetadata"
)

// FIXME: This reader should filter its input lines based on the search query
// from the pager.

type FilteringReader struct {
	backingReader Reader

	// This is a reference to a reference so that we can track changes to the
	// original pattern, including if it is set to nil.
	filterPattern **regexp.Regexp
}

func (f FilteringReader) getAllLines() []*NumberedLine {
	lines := make([]*NumberedLine, 0)

	allBaseLines := f.backingReader.GetLines(linemetadata.Index{}, math.MaxInt)
	resultIndex := 0
	filterPattern := *f.filterPattern
	for _, line := range allBaseLines.lines {
		if filterPattern != nil && len(filterPattern.String()) > 0 && !filterPattern.MatchString(line.line.Plain(&line.index)) {
			// We have a pattern but it doesn't match
			continue
		}

		lines = append(lines, &NumberedLine{
			line:   line.line,
			index:  linemetadata.IndexFromZeroBased(resultIndex),
			number: line.number,
		})
		resultIndex++
	}

	return lines
}

func (f FilteringReader) GetLineCount() int {
	return len(f.getAllLines())
}

func (f FilteringReader) GetLine(index linemetadata.Index) *NumberedLine {
	allLines := f.getAllLines()
	if index.Index() < 0 || index.Index() >= len(allLines) {
		return nil
	}
	return allLines[index.Index()]
}

func (f FilteringReader) GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	lines := f.getAllLines()

	if len(lines) == 0 || wantedLineCount == 0 {
		return &InputLines{
			statusText: "No filtered lines",
		}
	}

	lastLine := firstLine.NonWrappingAdd(wantedLineCount - 1)

	// Prevent reading past the end of the available lines
	maxLineNumber := *linemetadata.IndexFromLength(len(lines))
	if lastLine.IsAfter(maxLineNumber) {
		lastLine = maxLineNumber

		// If one line was requested, then first and last should be exactly the
		// same, and we would get there by adding zero.
		firstLine = lastLine.NonWrappingAdd(1 - wantedLineCount)

		return f.GetLines(firstLine, firstLine.CountLinesTo(lastLine))
	}

	return &InputLines{
		lines:      lines[firstLine.Index() : firstLine.Index()+wantedLineCount],
		statusText: "Filtered lines",
	}
}
