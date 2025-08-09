package linemetadata

import (
	"fmt"
	"math"

	"github.com/walles/moor/internal/util"
)

// This represents a line number in an input stream
type Number struct {
	number int
}

func (l Number) AsOneBased() int {
	if l.number == math.MaxInt {
		return math.MaxInt
	}

	return l.number + 1
}

// FIXME: Maybe drop this in favor of some array access method(s)?
func (l Number) AsZeroBased() int {
	return l.number
}

func NumberFromOneBased(oneBased int) Number {
	if oneBased < 1 {
		panic(fmt.Errorf("one-based line numbers must be at least 1, got %d", oneBased))
	}
	return Number{number: oneBased - 1}
}

func NumberFromZeroBased(zeroBased int) Number {
	if zeroBased < 0 {
		panic(fmt.Errorf("zero-based line numbers must be at least 0, got %d", zeroBased))
	}
	return Number{number: zeroBased}
}

// The highest possible line number
func NumberMax() Number {
	return Number{number: math.MaxInt}
}

// Set the line number to the last line of a file with the given number of lines
// in it. Or nil if the line count is 0.
func NumberFromLength(length int) *Number {
	if length == 0 {
		return nil
	}
	if length < 0 {
		panic(fmt.Errorf("line count must be at least 0, got %d", length))
	}
	return &Number{number: length - 1}
}

func (l Number) NonWrappingAdd(offset int) Number {
	if offset > 0 {
		if l.AsZeroBased() > math.MaxInt-offset {
			return NumberFromZeroBased(math.MaxInt)
		}
	} else {
		if l.AsZeroBased() < -offset {
			return NumberFromZeroBased(0)
		}
	}

	return NumberFromZeroBased(l.number + offset)
}

func (l Number) Format() string {
	return util.FormatInt(l.AsOneBased())
}

// If both lines are the same this method will return 1.
func (l Number) CountLinesTo(next Number) int {
	if l.number > next.number {
		panic(fmt.Errorf("line numbers must be ordered, got %s-%s", l.Format(), next.Format()))
	}

	return 1 + next.AsZeroBased() - l.AsZeroBased()
}

// Is this the lowest possible line number?
func (l Number) IsZero() bool {
	return l.AsZeroBased() == 0
}

func (l Number) IsBefore(other Number) bool {
	return l.AsZeroBased() < other.AsZeroBased()
}

func (l Number) IsAfter(other Number) bool {
	return l.AsZeroBased() > other.AsZeroBased()
}
