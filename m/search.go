package m

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/walles/moar/m/linenumbers"
)

func (p *Pager) scrollToSearchHits() {
	if p.searchPattern == nil {
		// This is not a search
		return
	}

	lineNumber := p.scrollPosition.lineNumber(p)
	if lineNumber == nil {
		// No lines to search
		return
	}

	firstHitPosition := p.findFirstHit(*lineNumber, nil, false)
	if firstHitPosition == nil && (*lineNumber != linenumbers.LineNumber{}) {
		// Try again from the top
		firstHitPosition = p.findFirstHit(linenumbers.LineNumber{}, lineNumber, false)
	}
	if firstHitPosition == nil {
		// No match, give up
		return
	}

	if firstHitPosition.isVisible(p) {
		// Already on-screen, never mind
		return
	}

	p.scrollPosition = *firstHitPosition
}

// NOTE: When we search, we do that by looping over the *input lines*, not the
// screen lines. That's why startPosition is a LineNumber rather than a
// scrollPosition.
//
// The `beforePosition` parameter is exclusive, meaning that line will not be
// searched.
//
// For the actual searching, this method will call _findFirstHit() in parallel
// on multiple cores, to help large file search performance.
//
// FIXME: We should take startPosition.deltaScreenLines into account as well!
func (p *Pager) findFirstHit(startPosition linenumbers.LineNumber, beforePosition *linenumbers.LineNumber, backwards bool) *scrollPosition {
	// If the number of lines to search matches the number of cores (or more),
	// divide the search into chunks. Otherwise use one chunk.
	chunkCount := runtime.NumCPU()
	var linesCount int
	if backwards {
		// If the startPosition is zero, that should make the count one
		linesCount = startPosition.AsZeroBased() + 1
		if beforePosition != nil {
			// Searching from 1 with before set to 0 should make the count 1
			linesCount = startPosition.AsZeroBased() - beforePosition.AsZeroBased()
		}
	} else {
		linesCount = p.reader.GetLineCount() - startPosition.AsZeroBased()
		if beforePosition != nil {
			// Searching from 1 with before set to 2 should make the count 1
			linesCount = beforePosition.AsZeroBased() - startPosition.AsZeroBased()
		}
	}

	if linesCount < chunkCount {
		chunkCount = 1
	}
	chunkSize := linesCount / chunkCount

	log.Debugf("Searching %d lines across %d cores with %d lines per core...", linesCount, chunkCount, chunkSize)
	t0 := time.Now()
	defer func() {
		linesPerSecond := float64(linesCount) / time.Since(t0).Seconds()
		linesPerSecondS := fmt.Sprintf("%.0f", linesPerSecond)
		if linesPerSecond > 7_000_000.0 {
			linesPerSecondS = fmt.Sprintf("%.0fM", linesPerSecond/1000_000.0)
		} else if linesPerSecond > 7_000.0 {
			linesPerSecondS = fmt.Sprintf("%.0fk", linesPerSecond/1000.0)
		}

		if linesCount > 0 {
			log.Debugf("Searched %d lines in %s at %slines/s or %s/line",
				linesCount,
				time.Since(t0),
				linesPerSecondS,
				time.Since(t0)/time.Duration(linesCount))
		} else {
			log.Debugf("Searched %d lines in %s at %slines/s", linesCount, time.Since(t0), linesPerSecondS)
		}
	}()

	// Each parallel search will start at one of these positions
	searchStarts := make([]linenumbers.LineNumber, chunkCount)
	direction := 1
	if backwards {
		direction = -1
	}
	for i := 0; i < chunkCount; i++ {
		searchStarts[i] = startPosition.NonWrappingAdd(i * direction * chunkSize)
	}

	// Make a results array, with one result per chunk
	findings := make([]chan *scrollPosition, chunkCount)

	// Search all chunks in parallel
	for i, searchStart := range searchStarts {
		findings[i] = make(chan *scrollPosition)

		searchEndIndex := i + 1
		var chunkBefore *linenumbers.LineNumber
		if searchEndIndex < len(searchStarts) {
			chunkBefore = &searchStarts[searchEndIndex]
		} else if beforePosition != nil {
			chunkBefore = beforePosition
		}

		go func(i int, searchStart linenumbers.LineNumber, chunkBefore *linenumbers.LineNumber) {
			defer func() {
				panicHandler("findFirstHit()/chunkSearch", recover(), debug.Stack())
			}()

			findings[i] <- p._findFirstHit(searchStart, chunkBefore, backwards)
		}(i, searchStart, chunkBefore)
	}

	// Return the first non-nil result
	for _, finding := range findings {
		result := <-finding
		if result != nil {
			return result
		}
	}

	return nil
}

// NOTE: When we search, we do that by looping over the *input lines*, not the
// screen lines. That's why startPosition is a LineNumber rather than a
// scrollPosition.
//
// The `beforePosition` parameter is exclusive, meaning that line will not be
// searched.
//
// This method will run over multiple chunks of the input file in parallel to
// help large file search performance.
//
// FIXME: We should take startPosition.deltaScreenLines into account as well!
func (p *Pager) _findFirstHit(startPosition linenumbers.LineNumber, beforePosition *linenumbers.LineNumber, backwards bool) *scrollPosition {
	searchPosition := startPosition
	for {
		line := p.reader.GetLine(searchPosition)
		if line == nil {
			// No match, give up
			return nil
		}

		lineText := line.Plain(&searchPosition)
		if p.searchPattern.MatchString(lineText) {
			return scrollPositionFromLineNumber("findFirstHit", searchPosition)
		}

		if backwards {
			if (searchPosition == linenumbers.LineNumber{}) {
				// Reached the top without any match, give up
				return nil
			}

			searchPosition = searchPosition.NonWrappingAdd(-1)
		} else {
			searchPosition = searchPosition.NonWrappingAdd(1)
		}

		if beforePosition != nil && searchPosition == *beforePosition {
			// No match, give up
			return nil
		}
	}
}

func (p *Pager) isViewing() bool {
	_, isViewing := p.mode.(PagerModeViewing)
	return isViewing
}

func (p *Pager) isNotFound() bool {
	_, isNotFound := p.mode.(PagerModeNotFound)
	return isNotFound
}

func (p *Pager) scrollToNextSearchHit() {
	if p.searchPattern == nil {
		// Nothing to search for, never mind
		return
	}

	if p.reader.GetLineCount() == 0 {
		// Nothing to search in, never mind
		return
	}

	if p.isViewing() && p.isScrolledToEnd() {
		p.mode = PagerModeNotFound{pager: p}
		return
	}

	var firstSearchPosition linenumbers.LineNumber

	switch {
	case p.isViewing():
		// Start searching on the first line below the bottom of the screen
		position := p.getLastVisiblePosition().NextLine(1)
		firstSearchPosition = *position.lineNumber(p)

	case p.isNotFound():
		// Restart searching from the top
		p.mode = PagerModeViewing{pager: p}
		firstSearchPosition = linenumbers.LineNumber{}

	default:
		panic(fmt.Sprint("Unknown search mode when finding next: ", p.mode))
	}

	firstHitPosition := p.findFirstHit(firstSearchPosition, nil, false)
	if firstHitPosition == nil {
		p.mode = PagerModeNotFound{pager: p}
		return
	}
	p.scrollPosition = *firstHitPosition

	// Don't let any search hit scroll out of sight
	p.TargetLineNumber = nil
}

func (p *Pager) scrollToPreviousSearchHit() {
	if p.searchPattern == nil {
		// Nothing to search for, never mind
		return
	}

	if p.reader.GetLineCount() == 0 {
		// Nothing to search in, never mind
		return
	}

	var firstSearchPosition linenumbers.LineNumber

	switch {
	case p.isViewing():
		// Start searching on the first line above the top of the screen
		position := p.scrollPosition.PreviousLine(1)
		firstSearchPosition = *position.lineNumber(p)

	case p.isNotFound():
		// Restart searching from the bottom
		p.mode = PagerModeViewing{pager: p}
		firstSearchPosition = *linenumbers.LineNumberFromLength(p.reader.GetLineCount())

	default:
		panic(fmt.Sprint("Unknown search mode when finding previous: ", p.mode))
	}

	firstHitPosition := p.findFirstHit(firstSearchPosition, nil, true)
	if firstHitPosition == nil {
		p.mode = PagerModeNotFound{pager: p}
		return
	}
	p.scrollPosition = *firstHitPosition

	// Don't let any search hit scroll out of sight
	p.TargetLineNumber = nil
}
