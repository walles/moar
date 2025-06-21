package m

import "github.com/walles/moar/m/linemetadata"

// FIXME: This reader should filter its input lines based on the search query
// from the pager.
//
// For starters, let's just make it always return the last line and see what happens.

type FilteringReader struct {
	backingReader Reader
}

func (f FilteringReader) GetLineCount() int {
	return min(f.backingReader.GetLineCount(), 1)
}

func (f FilteringReader) GetLine(index linemetadata.Index) *NumberedLine {
	if index.Index() != 0 {
		return nil
	}
	if f.backingReader.GetLineCount() == 0 {
		return nil
	}
	return f.backingReader.GetLine(linemetadata.IndexFromZeroBased(f.backingReader.GetLineCount() - 1))
}


This code crashes if you do "./moar.sh /etc/services", press & and then resize the window.


func (f FilteringReader) GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	if firstLine.Index() != 0 || f.backingReader.GetLineCount() == 0 {
		return &InputLines{
			statusText: "FIXME: This is a dummy filter to demonstrate the interface",
		}
	}

	result := f.backingReader.GetLines(linemetadata.IndexFromZeroBased(f.backingReader.GetLineCount()-1), min(wantedLineCount, 1))
	if len(result.lines) == 0 {
		return &InputLines{
			statusText: "Dummy filter found no lines",
		}
	}

	line := result.lines[0]
	line.index = linemetadata.IndexFromZeroBased(0)
	return &InputLines{
		lines:      []*NumberedLine{line},
		statusText: "FIXME: This is a dummy filter to demonstrate the interface",
		firstLine:  linemetadata.IndexFromZeroBased(0),
	}
}
