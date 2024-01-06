package m

import (
	"fmt"
	"math"
)

type LineNumber struct {
	number int
}

func (l LineNumber) AsOneBased() int {
	if l.number == math.MaxInt {
		return math.MaxInt
	}

	return l.number + 1
}

func (l LineNumber) AsZeroBased() int {
	return l.number
}

func LineNumberFromOneBased(oneBased int) LineNumber {
	return LineNumber{number: oneBased - 1}
}

func LineNumberFromZeroBased(zeroBased int) LineNumber {
	return LineNumber{number: zeroBased}
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
