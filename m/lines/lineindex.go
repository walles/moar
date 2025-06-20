package lines

import (
	"fmt"
	"math"
)

// This represents a (zero based) index into an array of lines
type Index struct {
	index int
}

func (i Index) Index() int {
	return i.index
}

// Set the line index to the last line of a file with the given number of lines
// in it. Or nil if the line count is 0.
func LineIndexFromLength(length int) *Index {
	if length == 0 {
		return nil
	}
	if length < 0 {
		panic(fmt.Errorf("line count must be at least 0, got %d", length))
	}
	return &Index{index: length - 1}
}

func (i Index) NonWrappingAdd(offset int) Index {
	if offset > 0 {
		if i.index > math.MaxInt-offset {
			return Index{index: math.MaxInt}
		}
	} else {
		if i.index < -offset {
			return Index{index: 0}
		}
	}

	return Index{index: i.index + offset}
}

func (i Index) Format() string {
	return formatInt(i.index + 1)
}

func (i Index) IsBefore(other Index) bool {
	return i.index < other.index
}

func (i Index) IsAfter(other Index) bool {
	return i.index > other.index
}

func (i Index) IsZero() bool {
	return i.index == 0
}
