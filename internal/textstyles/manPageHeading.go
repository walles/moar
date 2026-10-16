package textstyles

import (
	"unicode"
	"unicode/utf8"

	"github.com/walles/twin"
)

func manPageHeadingFromString(s string) *StyledRunesWithTrailer {
	// For great performance, first check the string without allocating any
	// memory.
	if !parseManPageHeading(s, func(_ CellWithMetadata) {}) {
		return nil
	}

	cells := make([]CellWithMetadata, 0, len(s)/2)
	ok := parseManPageHeading(s, func(cell CellWithMetadata) {
		cells = append(cells, cell)
	})
	if !ok {
		panic("man page heading state changed")
	}

	return &StyledRunesWithTrailer{
		StyledRunes: cells,
		Trailer:     twin.StyleDefault,
	}
}

// Reports back one cell at a time. Returns true if the entire string was a man
// page heading.
//
// If it was not, false will be returned and the cell reporting will be
// interrupted.
//
// A man page heading is all caps, and each char is man page bold as parsed by
// consumeBold(), including emboldened repeats. Whitespace is an exception, it
// can be not bold.
func parseManPageHeading(s string, reportStyledRune func(CellWithMetadata)) bool {
	if len(s) < 3 {
		// We don't want to match empty strings. Also, strings of length 1 and 2
		// cannot be man page headings since "char+backspace+char" is 3 bytes.
		return false
	}

	// For performance, reject most lines before getting into the details
	firstChar, firstCharSize := utf8.DecodeRuneInString(s)
	if unicode.IsSpace(firstChar) {
		// Headings do not start with whitespace
		return false
	}
	if firstCharSize >= len(s) || s[firstCharSize] != '\b' {
		// Headings start with a bold char
		return false
	}

	for runes := (lazyRunes{str: s}); runes.getRelative(0) != nil; runes.next() {
		bold := consumeEmboldened(&runes, consumeBold)
		if bold == nil {
			char := *runes.getRelative(0)
			if !unicode.IsSpace(char) {
				// Not bold and not whitespace => Not a heading
				return false
			}

			// Whitespace is an exception, it can be not bold
			reportStyledRune(CellWithMetadata{Rune: char, Style: ManPageHeading})
			continue
		}

		if unicode.IsLetter(bold.Rune) && !unicode.IsUpper(bold.Rune) {
			// Not ALL CAPS => Not a heading
			return false
		}

		reportStyledRune(CellWithMetadata{Rune: bold.Rune, Style: ManPageHeading})
	}

	return true
}
