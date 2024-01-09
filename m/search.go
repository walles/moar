package m

import (
	"fmt"
)

func (p *Pager) scrollToSearchHits() {
	if p.searchPattern == nil {
		// This is not a search
		return
	}

	firstHitPosition := p.findFirstHit(p.scrollPosition, false)
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

func (p *Pager) findFirstHit(startPosition scrollPosition, backwards bool) *scrollPosition {
	// FIXME: We should take startPosition.deltaScreenLines into account as well!

	// NOTE: When we search, we do that by looping over the *input lines*, not
	// the screen lines. That's why we're using an int rather than a
	// scrollPosition for searching.
	searchPosition := *startPosition.lineNumber(p)
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
			searchPosition = searchPosition.NonWrappingAdd(-1)
		} else {
			searchPosition = searchPosition.NonWrappingAdd(1)
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

	var firstSearchPosition scrollPosition

	switch {
	case p.isViewing():
		// Start searching on the first line below the bottom of the screen
		firstSearchPosition = p.getLastVisiblePosition().NextLine(1)

	case p.isNotFound():
		// Restart searching from the top
		p.mode = PagerModeViewing{pager: p}
		firstSearchPosition = newScrollPosition("firstSearchPosition")

	default:
		panic(fmt.Sprint("Unknown search mode when finding next: ", p.mode))
	}

	firstHitPosition := p.findFirstHit(firstSearchPosition, false)
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

	var firstSearchPosition scrollPosition

	switch {
	case p.isViewing():
		// Start searching on the first line above the top of the screen
		firstSearchPosition = p.scrollPosition.PreviousLine(1)

	case p.isNotFound():
		// Restart searching from the bottom
		p.mode = PagerModeViewing{pager: p}
		p.scrollToEnd()

	default:
		panic(fmt.Sprint("Unknown search mode when finding previous: ", p.mode))
	}

	firstHitPosition := p.findFirstHit(firstSearchPosition, true)
	if firstHitPosition == nil {
		p.mode = PagerModeNotFound{pager: p}
		return
	}
	p.scrollPosition = *firstHitPosition

	// Don't let any search hit scroll out of sight
	p.TargetLineNumber = nil
}
