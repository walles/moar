package textstyles

import (
	"github.com/walles/moar/twin"
)

func manPageHeadingFromString(s string) *CellsWithTrailer {
	if !parseManPageHeading(s, func(_ twin.Cell) {}) {
		return nil
	}

	cells := make([]twin.Cell, 0, len(s)/2)
	ok := parseManPageHeading(s, func(cell twin.Cell) {
		cells = append(cells, cell)
	})
	if !ok {
		panic("man page heading state changed")
	}

	return &CellsWithTrailer{
		Cells:   cells,
		Trailer: twin.StyleDefault,
	}
}

// Reports back one cell at a time. Returns true if the entire string was a man
// page heading.
//
// If it was not, false will be returned and the cell reporting will be
// interrupted.
//
// A man page heading is all caps. Also, each character is encoded as
// char+backspace+char, where both chars need to be the same. Whitespace is an
// exception, they can be not bold.
func parseManPageHeading(s string, reportCell func(twin.Cell)) bool {
	if len(s) < 3 {
		// We don't want to match empty strings. Also, strings of length 1 and 2
		// cannot be man page headings since "char+backspace+char" is 3 bytes.
		return false
	}

	Johan: Write code here
}
