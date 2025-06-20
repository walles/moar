package lines

import (
	"math"
	"testing"

	"gotest.tools/v3/assert"
)

func TestNonWrappingAddPositive(t *testing.T) {
	base := NumberFromZeroBased(math.MaxInt - 2)
	assert.Equal(t, base.NonWrappingAdd(1), NumberFromZeroBased(math.MaxInt-1))
	assert.Equal(t, base.NonWrappingAdd(2), NumberFromZeroBased(math.MaxInt))
	assert.Equal(t, base.NonWrappingAdd(3), NumberFromZeroBased(math.MaxInt))
}

func TestNonWrappingAddNegative(t *testing.T) {
	base := NumberFromZeroBased(2)
	assert.Equal(t, base.NonWrappingAdd(-1), NumberFromZeroBased(1))
	assert.Equal(t, base.NonWrappingAdd(-2), NumberFromZeroBased(0))
	assert.Equal(t, base.NonWrappingAdd(-3), NumberFromZeroBased(0))
}

func TestLineNumberFormatting(t *testing.T) {
	assert.Equal(t, "1", NumberFromOneBased(1).Format())
	assert.Equal(t, "10", NumberFromOneBased(10).Format())
	assert.Equal(t, "100", NumberFromOneBased(100).Format())

	// Ref: // https://en.wikipedia.org/wiki/Decimal_separator#Exceptions_to_digit_grouping
	assert.Equal(t, "1000", NumberFromOneBased(1000).Format())

	assert.Equal(t, "10_000", NumberFromOneBased(10000).Format())
	assert.Equal(t, "100_000", NumberFromOneBased(100000).Format())
	assert.Equal(t, "1_000_000", NumberFromOneBased(1000000).Format())
	assert.Equal(t, "10_000_000", NumberFromOneBased(10000000).Format())
}

func TestLineNumberFromLength(t *testing.T) {
	// If the file has one line then the last zero based line number is 0.
	fromLength := LineNumberFromLength(1)
	assert.Equal(t, *fromLength, NumberFromZeroBased(0))
}

func TestLineNumberEquality(t *testing.T) {
	assert.Equal(t, NumberFromZeroBased(1), NumberFromOneBased(2),
		"Two different ways of representing the same line number should be equal")
}

func TestLineNumberCountLinesTo(t *testing.T) {
	assert.Equal(t,
		NumberFromZeroBased(0).CountLinesTo(NumberFromZeroBased(0)),
		1, // Count is inclusive, so countint from 0 to 0 is 1
	)

	assert.Equal(t,
		NumberFromZeroBased(0).CountLinesTo(NumberFromZeroBased(1)),
		2, // Count is inclusive, so countint from 0 to 1 is 2
	)
}
