package linemetadata

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

func IndexFromOneBased(oneBased int) Index {
	if oneBased < 1 {
		panic(fmt.Errorf("one-based line indices must be at least 1, got %d", oneBased))
	}
	return Index{index: oneBased - 1}
}

// The highest possible line index
func IndexMax() Index {
	return Index{index: math.MaxInt}
}

// Set the line index to the last line of a file with the given number of lines
// in it. Or nil if the line count is 0.
func IndexFromLength(length int) *Index {
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

// If both indices are the same this method will return 1.
func (i Index) CountLinesTo(next Index) int {
	if i.index > next.index {
		panic(fmt.Errorf("line indices must be ordered, got %s-%s", i.Format(), next.Format()))
	}

	return 1 + next.index - i.index
}

func (i Index) IsZero() bool {
	return i.index == 0
}

func (i Index) IsWithinLength(length int) bool {
	if length < 0 {
		panic(fmt.Errorf("line count must be at least 0, got %d", length))
	}
	return i.index >= 0 && i.index < length
}
