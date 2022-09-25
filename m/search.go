package m

import (
	"fmt"
	"regexp"
	"unicode"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/twin"
)

func (p *Pager) addSearchFooter() {
	width, height := p.screen.Size()

	pos := 0
	for _, token := range "Search: " + p.searchString {
		p.screen.SetCell(pos, height-1, twin.NewCell(token, twin.StyleDefault))
		pos++
	}

	// Add a cursor
	p.screen.SetCell(pos, height-1, twin.NewCell(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))
	pos++

	// Clear the rest of the line
	for pos < width {
		p.screen.SetCell(pos, height-1, twin.NewCell(' ', twin.StyleDefault))
		pos++
	}
}

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

	if p.isVisible(*firstHitPosition) {
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
	searchPosition := startPosition.lineNumberOneBased(p)
	for {
		line := p.reader.GetLine(searchPosition)
		if line == nil {
			// No match, give up
			return nil
		}

		lineText := line.Plain()
		if p.searchPattern.MatchString(lineText) {
			return scrollPositionFromLineNumber("findFirstHit", searchPosition)
		}

		if backwards {
			searchPosition -= 1
		} else {
			searchPosition += 1
		}
	}
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

	if p.mode == _Viewing && p.isScrolledToEnd() {
		p.mode = _NotFound
		return
	}

	var firstSearchPosition scrollPosition

	switch p.mode {
	case _Viewing:
		// Start searching on the first line below the bottom of the screen
		firstSearchPosition = p.getLastVisiblePosition().NextLine(1)

	case _NotFound:
		// Restart searching from the top
		p.mode = _Viewing
		firstSearchPosition = newScrollPosition("firstSearchPosition")

	default:
		panic(fmt.Sprint("Unknown search mode when finding next: ", p.mode))
	}

	firstHitPosition := p.findFirstHit(firstSearchPosition, false)
	if firstHitPosition == nil {
		p.mode = _NotFound
		return
	}
	p.scrollPosition = *firstHitPosition
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

	switch p.mode {
	case _Viewing:
		// Start searching on the first line above the top of the screen
		firstSearchPosition = p.scrollPosition.PreviousLine(1)

	case _NotFound:
		// Restart searching from the bottom
		p.mode = _Viewing
		p.scrollToEnd()

	default:
		panic(fmt.Sprint("Unknown search mode when finding previous: ", p.mode))
	}

	firstHitPosition := p.findFirstHit(firstSearchPosition, true)
	if firstHitPosition == nil {
		p.mode = _NotFound
		return
	}
	p.scrollPosition = *firstHitPosition
}

func (p *Pager) updateSearchPattern() {
	p.searchPattern = toPattern(p.searchString)

	p.scrollToSearchHits()

	// FIXME: If the user is typing, indicate to user if we didn't find anything
}

// toPattern compiles a search string into a pattern.
//
// If the string contains only lower-case letter the pattern will be case insensitive.
//
// If the string is empty the pattern will be nil.
//
// If the string does not compile into a regexp the pattern will match the string verbatim
func toPattern(compileMe string) *regexp.Regexp {
	if len(compileMe) == 0 {
		return nil
	}

	hasUppercase := false
	for _, char := range compileMe {
		if unicode.IsUpper(char) {
			hasUppercase = true
		}
	}

	// Smart case; be case insensitive unless there are upper case chars
	// in the search string
	prefix := "(?i)"
	if hasUppercase {
		prefix = ""
	}

	pattern, err := regexp.Compile(prefix + compileMe)
	if err == nil {
		// Search string is a regexp
		return pattern
	}

	pattern, err = regexp.Compile(prefix + regexp.QuoteMeta(compileMe))
	if err == nil {
		// Pattern matching the string exactly
		return pattern
	}

	// Unable to create a match-string-verbatim pattern
	panic(err)
}

// From: https://stackoverflow.com/a/57005674/473672
func removeLastChar(s string) string {
	r, size := utf8.DecodeLastRuneInString(s)
	if r == utf8.RuneError && (size == 0 || size == 1) {
		size = 0
	}
	return s[:len(s)-size]
}

func (p *Pager) onSearchKey(key twin.KeyCode) {
	switch key {
	case twin.KeyEscape, twin.KeyEnter:
		p.mode = _Viewing

	case twin.KeyBackspace, twin.KeyDelete:
		if len(p.searchString) == 0 {
			return
		}

		p.searchString = removeLastChar(p.searchString)
		p.updateSearchPattern()

	case twin.KeyUp:
		// Clipping is done in _Redraw()
		p.scrollPosition = p.scrollPosition.PreviousLine(1)
		p.mode = _Viewing

	case twin.KeyDown:
		// Clipping is done in _Redraw()
		p.scrollPosition = p.scrollPosition.NextLine(1)
		p.mode = _Viewing

	case twin.KeyPgUp:
		_, height := p.screen.Size()
		p.scrollPosition = p.scrollPosition.PreviousLine(height - 1)
		p.mode = _Viewing

	case twin.KeyPgDown:
		_, height := p.screen.Size()
		p.scrollPosition = p.scrollPosition.NextLine(height - 1)
		p.mode = _Viewing

	default:
		log.Debugf("Unhandled search key event %v", key)
	}
}

func (p *Pager) onSearchRune(char rune) {
	p.searchString = p.searchString + string(char)
	p.updateSearchPattern()
}
