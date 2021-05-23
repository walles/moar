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
