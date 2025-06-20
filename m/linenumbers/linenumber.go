package linenumbers

import (
	"fmt"
	"math"
)

// This represents a line number in an input stream
type LineNumber struct {
	number int
}

func (l LineNumber) AsOneBased() int {
	if l.number == math.MaxInt {
		return math.MaxInt
	}

	return l.number + 1
}

// FIXME: Maybe drop this in favor of some array access method(s)?
func (l LineNumber) AsZeroBased() int {
	return l.number
}

func LineNumberFromOneBased(oneBased int) LineNumber {
	if oneBased < 1 {
		panic(fmt.Errorf("one-based line numbers must be at least 1, got %d", oneBased))
	}
	return LineNumber{number: oneBased - 1}
}

func LineNumberFromZeroBased(zeroBased int) LineNumber {
	if zeroBased < 0 {
		panic(fmt.Errorf("zero-based line numbers must be at least 0, got %d", zeroBased))
	}
	return LineNumber{number: zeroBased}
}

// The highest possible line number
func LineNumberMax() LineNumber {
	return LineNumber{number: math.MaxInt}
}

// Set the line number to the last line of a file with the given number of lines
// in it. Or nil if the line count is 0.
func LineNumberFromLength(length int) *LineNumber {
	if length == 0 {
		return nil
	}
	if length < 0 {
		panic(fmt.Errorf("line count must be at least 0, got %d", length))
	}
	return &LineNumber{number: length - 1}
}

func (l LineNumber) NonWrappingAdd(offset int) LineNumber {
	if offset > 0 {
		if l.AsZeroBased() > math.MaxInt-offset {
			return LineNumberFromZeroBased(math.MaxInt)
		}
	} else {
		if l.AsZeroBased() < -offset {
			return LineNumberFromZeroBased(0)
		}
	}

	return LineNumberFromZeroBased(l.number + offset)
}

// Formats a number into a string with _ between each three-group of digits, for
// numbers >= 10_000.
//
// Regarding the >= 10_000 exception:
// https://en.wikipedia.org/wiki/Decimal_separator#Exceptions_to_digit_grouping
func (l LineNumber) Format() string {
	if l.AsOneBased() < 10_000 {
		return fmt.Sprint(l.AsOneBased())
	}

	result := ""

	chars := []rune(fmt.Sprint(l.AsOneBased()))
	addCount := 0
	for i := len(chars) - 1; i >= 0; i-- {
		char := chars[i]
		if len(result) > 0 && addCount%3 == 0 {
			result = "_" + result
		}
		result = string(char) + result
		addCount++
	}

	return result
}

// If both lines are the same this method will return 1.
func (l LineNumber) CountLinesTo(next LineNumber) int {
	if l.number > next.number {
		panic(fmt.Errorf("line numbers must be ordered, got %s-%s", l.Format(), next.Format()))
	}

	return 1 + next.AsZeroBased() - l.AsZeroBased()
}

// Is this the lowest possible line number?
func (l LineNumber) IsZero() bool {
	return l.AsZeroBased() == 0
}

func (l LineNumber) IsBefore(other LineNumber) bool {
	return l.AsZeroBased() < other.AsZeroBased()
}

func (l LineNumber) IsAfter(other LineNumber) bool {
	return l.AsZeroBased() > other.AsZeroBased()
}
