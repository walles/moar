package m

import (
	"regexp"
	"sync"
	"unicode"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/twin"
)

type searchCommand int

const (
	searchCommandSearch searchCommand = iota
	searchCommandDone
)

type PagerModeSearch struct {
	pager *Pager

	pattern   *regexp.Regexp
	startLine linenumbers.LineNumber
	lock      *sync.Mutex

	searcher chan searchCommand
}

func (m PagerModeSearch) drawFooter(_ string, _ string) {
	width, height := m.pager.screen.Size()

	pos := 0
	for _, token := range "Search: " + m.pager.searchString {
		m.pager.screen.SetCell(pos, height-1, twin.NewCell(token, twin.StyleDefault))
		pos++
	}

	// Add a cursor
	m.pager.screen.SetCell(pos, height-1, twin.NewCell(' ', twin.StyleDefault.WithAttr(twin.AttrReverse)))
	pos++

	// Clear the rest of the line
	for pos < width {
		m.pager.screen.SetCell(pos, height-1, twin.NewCell(' ', twin.StyleDefault))
		pos++
	}
}

func (m PagerModeSearch) searcherSearch() *linenumbers.LineNumber {
	// Search to the end
	for position := m.startLine; ; position = position.NonWrappingAdd(1) {
		line := m.pager.reader.GetLine(position)
		if line == nil {
			// Reached end of input without any match, give up
			break
		}

		if m.pattern.MatchString(line.Plain(&position)) {
			return &position
		}
	}

	// Search from the beginning
	for position := m.startLine; position != m.startLine; position = position.NonWrappingAdd(1) {
		line := m.pager.reader.GetLine(position)
		if m.pattern.MatchString(line.Plain(&position)) {
			return &position
		}
	}

	return nil
}

func (m *PagerModeSearch) initSearcher() {
	m.lock = &sync.Mutex{}
	m.searcher = make(chan searchCommand, 1)

	go func() {
		for command := range m.searcher {
			switch command {
			case searchCommandSearch:
				found := m.searcherSearch()
				if found != nil {
					FIXME: Tell the pager to scroll to this position
				}
			case searchCommandDone:
				return
			}
		}
	}()
}

func (m *PagerModeSearch) updateSearchPattern() {
	// For highlighting
	m.pager.searchPattern = toPattern(m.pager.searchString)
	if m.pager.searchPattern == nil {
		// Nothing to search for, never mind
		return
	}
	startLine := m.pager.scrollPosition.lineNumber(m.pager)
	if startLine == nil {
		// Nothing to search in, never mind
		return
	}

	if m.searcher == nil {
		m.initSearcher()
	}

	// Give the searcher the new pattern
	m.lock.Lock()
	m.pattern = m.pager.searchPattern
	m.startLine = *startLine
	m.lock.Unlock()

	// Tell the searcher there's a new pattern to look for
	select {
	case m.searcher <- searchCommandSearch:
	default:
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
	case twin.KeyEscape, twin.KeyEnter:
		//nolint:gosimple // The linter's advice is just wrong here
		m.pager.mode = PagerModeViewing{pager: m.pager}

	case twin.KeyBackspace, twin.KeyDelete:
		if len(m.pager.searchString) == 0 {
			return
		}

		m.pager.searchString = removeLastChar(m.pager.searchString)
		m.updateSearchPattern()

	case twin.KeyUp, twin.KeyDown, twin.KeyPgUp, twin.KeyPgDown:
		//nolint:gosimple // The linter's advice is just wrong here
		m.pager.mode = PagerModeViewing{pager: m.pager}
		m.pager.mode.onKey(key)

	default:
		log.Debugf("Unhandled search key event %v", key)
	}
}

func (m PagerModeSearch) onRune(char rune) {
	m.pager.searchString = m.pager.searchString + string(char)
	m.updateSearchPattern()
}
