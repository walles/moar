package lineindices

import (
	"fmt"
	"math"
)

// This represents a (zero based) index into an array of lines
type LineIndex struct {
	index int
}

func (i LineIndex) IsZero() bool {
	return i.index == 0
}

func (i LineIndex) NonWrappingAdd(offset int) LineIndex {
	if offset > 0 {
		if i.index > math.MaxInt-offset {
			return LineIndex{index: math.MaxInt}
		}
	} else {
		if i.index < -offset {
			return LineIndex{index: 0}
		}
	}

	return LineIndex{index: i.index + offset}
}

// Set the line index to the last line of a file with the given number of lines
// in it. Or nil if the line count is 0.
func LineIndexFromLength(length int) *LineIndex {
	if length == 0 {
		return nil
	}
	if length < 0 {
		panic(fmt.Errorf("line count must be at least 0, got %d", length))
	}
	return &LineIndex{index: length - 1}
}

func (i LineIndex) IsBefore(other LineIndex) bool {
	return i.index < other.index
}

func (i LineIndex) IsAfter(other LineIndex) bool {
	return i.index > other.index
}
