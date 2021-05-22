package m

import "github.com/walles/moar/twin"

func wrapLine(width int, line []twin.Cell) [][]twin.Cell {
	if len(line) == 0 {
		return [][]twin.Cell{{}}
	}

	wrapped := make([][]twin.Cell, 0, len(line)/width)
	for len(line) > width {
		firstPart := line[:width]
		wrapped = append(wrapped, firstPart)

		line = line[width:]
	}

	if len(line) > 0 {
		wrapped = append(wrapped, line)
	}

	return wrapped
}
