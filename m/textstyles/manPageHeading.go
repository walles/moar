package textstyles

import (
	"unicode"

	"github.com/walles/moar/twin"
)

func manPageHeadingFromString(s string) *CellsWithTrailer {
	if !isManPageHeading(s) {
		return nil
	}

	cells := make([]twin.Cell, 0, len(s)/3)
	for i, char := range s {
		if i%3 > 0 {
			continue
		}

		cells = append(
			cells,
			twin.Cell{Rune: char, Style: ManPageHeading},
		)
	}

	return &CellsWithTrailer{
		Cells:   cells,
		Trailer: twin.StyleDefault,
	}
}

// A man page heading is all caps. Also, each character is encoded as
// char+backspace+char, where both chars need to be the same.
func isManPageHeading(s string) bool {
	if len(s) < 3 {
		// We don't want to match empty strings. Also, strings of length 1 and 2
		// cannot be man page headings since "char+backspace+char" is 3 bytes.
		return false
	}

	var currentChar rune
	nextCharNumber := 0
	for _, char := range s {
		currentCharNumber := nextCharNumber
		nextCharNumber++
		switch currentCharNumber % 3 {
		case 0:
			if !isManPageHeadingChar(char) {
				return false
			}
			currentChar = char
		case 1:
			if char != '\b' {
				return false
			}
		case 2:
			if char != currentChar {
				return false
			}
		default:
			panic("Impossible")
		}
	}

	return nextCharNumber%3 == 0
}

// Alphabetic chars must be upper case, all others are fine.
func isManPageHeadingChar(char rune) bool {
	if !unicode.IsLetter(char) {
		return true
	}

	return unicode.IsUpper(char)
}
