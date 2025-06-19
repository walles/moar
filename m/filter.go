package m

import "github.com/walles/moar/m/linenumbers"

func (p *Pager) GetFilteredLines() *InputLines {
	// FIXME: If we are filtering, get only the matching lines

	wantedLineCount := p.visibleHeight()

	var lineNumber linenumbers.LineNumber
	if p.lineNumber() != nil {
		lineNumber = *p.lineNumber()
	} else {
		// No lines to show, line number doesn't matter, pick anything. But we
		// still want one so that we can get the status text from the reader
		// below.
		lineNumber = linenumbers.LineNumber{}
	}

	return p.reader.GetLines(lineNumber, wantedLineCount)
}
