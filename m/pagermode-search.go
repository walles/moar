package m

import (
	"regexp"
	"unicode"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/twin"
)

type SearchDirection int

const (
	SearchDirectionForward SearchDirection = iota
	SearchDirectionBackward
)

type PagerModeSearch struct {
	pager                 *Pager
	initialScrollPosition scrollPosition // Pager position before search started
	direction             SearchDirection
}

func (m PagerModeSearch) drawFooter(_ string, _ string) {
	width, height := m.pager.screen.Size()

	prompt := "Search: "
	if m.direction == SearchDirectionBackward {
		prompt = "Search backwards: "
	}

	pos := 0
	for _, token := range prompt + m.pager.searchString {
		pos += m.pager.screen.SetCell(pos, height-1, twin.NewStyledRune(token, twin.StyleDefault))
	}

	// Add a cursor
	pos += m.pager.screen.SetCell(pos, height-1, twin.NewStyledRune(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))

	// Clear the rest of the line
	for pos < width {
		pos += m.pager.screen.SetCell(pos, height-1, twin.NewStyledRune(' ', twin.StyleDefault))
	}
}

func (m *PagerModeSearch) updateSearchPattern() {
	m.pager.searchPattern = toPattern(m.pager.searchString)

	switch m.direction {
	case SearchDirectionBackward:
		m.pager.scrollToSearchHitsBackwards()
	case SearchDirectionForward:
		m.pager.scrollToSearchHits()
	}
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

func (m PagerModeSearch) onKey(key twin.KeyCode) {
	switch key {
	case twin.KeyCtrlD:
		m.pager.Quit()
	case twin.KeyEnter:
		m.pager.mode = PagerModeViewing{pager: m.pager}

	case twin.KeyEscape:
		m.pager.mode = PagerModeViewing{pager: m.pager}
		m.pager.scrollPosition = m.initialScrollPosition

	case twin.KeyBackspace, twin.KeyDelete:
		if len(m.pager.searchString) == 0 {
			return
		}

		m.pager.searchString = removeLastChar(m.pager.searchString)
		m.updateSearchPattern()

	case twin.KeyUp, twin.KeyDown, twin.KeyPgUp, twin.KeyPgDown:
		m.pager.mode = PagerModeViewing{pager: m.pager}
		m.pager.mode.onKey(key)

	default:
		log.Debugf("Unhandled search key event %v", key)
	}
}

func (m PagerModeSearch) onRune(char rune) {
	if char == '\x08' {
		// Backspace
		if len(m.pager.searchString) == 0 {
			return
		}

		m.pager.searchString = removeLastChar(m.pager.searchString)
	} else {
		m.pager.searchString = m.pager.searchString + string(char)
	}

	m.updateSearchPattern()
}
