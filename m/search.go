package m

import (
	"fmt"

	"github.com/walles/moar/m/linenumbers"
)

func (p *Pager) scrollToSearchHits() {
	if p.searchPattern == nil {
		// This is not a search
		return
	}

	firstHitPosition := p.findFirstHit(*p.scrollPosition.lineNumber(p), nil, false)
	if firstHitPosition == nil {
		// Try again from the top
		firstHitPosition = p.findFirstHit(linenumbers.LineNumber{}, p.scrollPosition.lineNumber(p), false)
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

// NOTE: When we search, we do that by looping over the *input lines*, not
// the screen lines. That's why we're using a line number rather than a
// scrollPosition for searching.
//
// FIXME: We should take startPosition.deltaScreenLines into account as well!
func (p *Pager) findFirstHit(startPosition linenumbers.LineNumber, beforePosition *linenumbers.LineNumber, backwards bool) *scrollPosition {
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
				// No match, give up
				return nil
			}

			searchPosition = searchPosition.NonWrappingAdd(-1)
		} else {
			searchPosition = searchPosition.NonWrappingAdd(1)

			if beforePosition != nil && searchPosition == *beforePosition {
				// No match, give up
				return nil
			}
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
