package twin

import (
	"fmt"
	"unicode"

	"github.com/rivo/uniseg"
)

// StyledRune is a rune with a style to be written to a one or more cells on the
// screen. Note that a StyledRune may use more than one cell on the screen ('午'
// for example).
type StyledRune struct {
	Rune  rune
	Style Style
}

func NewStyledRune(rune rune, style Style) StyledRune {
	return StyledRune{
		Rune:  rune,
		Style: style,
	}
}

func (styledRune StyledRune) String() string {
	return fmt.Sprint("rune='", string(styledRune.Rune), "' ", styledRune.Style)
}

// How many screen cells will this rune cover? Most runes cover one, but some
// like '午' will cover two.
func (styledRune StyledRune) Width() int {
	return uniseg.StringWidth(string(styledRune.Rune))
}

// Returns a slice of cells with trailing whitespace cells removed
func TrimSpaceRight(runes []StyledRune) []StyledRune {
	for i := len(runes) - 1; i >= 0; i-- {
		cell := runes[i]
		if !unicode.IsSpace(cell.Rune) {
			return runes[0 : i+1]
		}

		// That was a space, keep looking
	}

	// All whitespace, return empty
	return []StyledRune{}
}

// Returns a slice of cells with leading whitespace cells removed
func TrimSpaceLeft(runes []StyledRune) []StyledRune {
	for i := 0; i < len(runes); i++ {
		cell := runes[i]
		if !unicode.IsSpace(cell.Rune) {
			return runes[i:]
		}

		// That was a space, keep looking
	}

	// All whitespace, return empty
	return []StyledRune{}
}

func Printable(char rune) bool {
	if unicode.IsPrint(char) {
		return true
	}

	if unicode.Is(unicode.Co, char) {
		// Co == "Private Use": https://www.compart.com/en/unicode/category
		//
		// This space is used by Font Awesome, for "fa-battery-empty" for
		// example: https://fontawesome.com/v4/icon/battery-empty
		//
		// So we want to print these and let the rendering engine deal with
		// outputting them in a way that's helpful to the user.
		return true
	}

	if char == 0xa0 {
		// 0xa0 is a non-breaking space, which is printable, despite what
		// unicode.IsPrint() says.
		return true
	}

	return false
}
