package m

import "github.com/walles/moar/m/linenumbers"

func (p *Pager) GetFilteredLines() *InputLines {
	var lineNumber linenumbers.LineNumber
	if p.lineNumber() != nil {
		lineNumber = *p.lineNumber()
	} else {
		// No lines to show, line number doesn't matter, pick anything. But we
		// still want one so that we can get the status text from the reader
		// below.
		lineNumber = linenumbers.LineNumber{}
	}

	if _, ok := p.mode.(*PagerModeFilter); !ok {
		return getFilteredLines(lineNumber, p.visibleHeight())
	}

	return p.reader.GetLines(lineNumber, p.visibleHeight())
}
