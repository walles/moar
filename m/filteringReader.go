package m

import (
	"math"
	"regexp"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/m/linemetadata"
)

// Filters lines based on the search query from the pager.

type FilteringReader struct {
	BackingReader Reader

	// This is a reference to a reference so that we can track changes to the
	// original pattern, including if it is set to nil.
	FilterPattern **regexp.Regexp

	// nil means no filtering has happened yet
	filteredLinesCache *[]*NumberedLine

	// This is what the reader's line count was when we filtered. If the
	// reader's current line count doesn't match, then our cache needs to be
	// rebuilt.
	unfilteredLineCountWhenCaching int

	// This is the pattern that was used when we cached the lines. If it
	// doesn't match the current pattern, then our cache needs to be rebuilt.
	filterPatternWhenCaching *regexp.Regexp
}

func (f *FilteringReader) rebuildCache() {
	t0 := time.Now()

	cache := make([]*NumberedLine, 0)
	filterPattern := *f.FilterPattern

	// Mark cache base conditions
	f.unfilteredLineCountWhenCaching = f.BackingReader.GetLineCount()
	f.filterPatternWhenCaching = filterPattern

	// Repopulate the cache
	allBaseLines := f.BackingReader.GetLines(linemetadata.Index{}, math.MaxInt)
	resultIndex := 0
	for _, line := range allBaseLines.lines {
		if filterPattern != nil && len(filterPattern.String()) > 0 && !filterPattern.MatchString(line.line.Plain(&line.index)) {
			// We have a pattern but it doesn't match
			continue
		}

		cache = append(cache, &NumberedLine{
			line:   line.line,
			index:  linemetadata.IndexFromZeroBased(resultIndex),
			number: line.number,
		})
		resultIndex++
	}

	f.filteredLinesCache = &cache

	log.Debugf("Filtered out %d/%d lines in %s",
		len(allBaseLines.lines)-len(cache), len(allBaseLines.lines), time.Since(t0))
}

func (f *FilteringReader) getAllLines() []*NumberedLine {
	if f.filteredLinesCache == nil {
		f.rebuildCache()
		return *f.filteredLinesCache
	}

	if f.unfilteredLineCountWhenCaching != f.BackingReader.GetLineCount() {
		f.rebuildCache()
		return *f.filteredLinesCache
	}

	var currentFilterPattern string
	if *f.FilterPattern != nil {
		currentFilterPattern = (*f.FilterPattern).String()
	}
	var cacheFilterPattern string
	if f.filterPatternWhenCaching != nil {
		cacheFilterPattern = f.filterPatternWhenCaching.String()
	}
	if currentFilterPattern != cacheFilterPattern {
		f.rebuildCache()
		return *f.filteredLinesCache
	}

	return *f.filteredLinesCache
}

func (f *FilteringReader) shouldPassThrough() bool {
	if *f.FilterPattern == nil || len((*f.FilterPattern).String()) == 0 {
		// Cache is not needed
		f.filteredLinesCache = nil

		// No filtering, so pass through all
		return true
	}

	return false
}

func (f *FilteringReader) GetLineCount() int {
	if f.shouldPassThrough() {
		return f.BackingReader.GetLineCount()
	}

	return len(f.getAllLines())
}

func (f *FilteringReader) GetLine(index linemetadata.Index) *NumberedLine {
	if f.shouldPassThrough() {
		return f.BackingReader.GetLine(index)
	}

	allLines := f.getAllLines()
	if index.Index() < 0 || index.Index() >= len(allLines) {
		return nil
	}
	return allLines[index.Index()]
}

func (f *FilteringReader) GetLines(firstLine linemetadata.Index, wantedLineCount int) *InputLines {
	if f.shouldPassThrough() {
		return f.BackingReader.GetLines(firstLine, wantedLineCount)
	}

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
