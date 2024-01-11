package twin

import (
	"fmt"
	"unicode"
)

// Cell is a rune with a style to be written to a cell on screen
type Cell struct {
	Rune  rune
	Style Style
}

func NewCell(rune rune, style Style) Cell {
	return Cell{
		Rune:  rune,
		Style: style,
	}
}

func (cell Cell) String() string {
	return fmt.Sprint("rune='", string(cell.Rune), "' ", cell.Style)
}

// Returns a slice of cells with trailing whitespace cells removed
func TrimSpaceRight(cells []Cell) []Cell {
	for i := len(cells) - 1; i >= 0; i-- {
		cell := cells[i]
		if !unicode.IsSpace(cell.Rune) {
			return cells[0 : i+1]
		}

		// That was a space, keep looking
	}

	// All whitespace, return empty
	return []Cell{}
}

// Returns a slice of cells with leading whitespace cells removed
func TrimSpaceLeft(cells []Cell) []Cell {
	for i := 0; i < len(cells); i++ {
		cell := cells[i]
		if !unicode.IsSpace(cell.Rune) {
			return cells[i:]
		}

		// That was a space, keep looking
	}

	// All whitespace, return empty
	return []Cell{}
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
