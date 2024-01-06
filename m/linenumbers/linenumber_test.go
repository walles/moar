package linenumbers

import (
	"math"
	"testing"

	"gotest.tools/v3/assert"
)

func TestNonWrappingAddPositive(t *testing.T) {
	base := LineNumberFromZeroBased(math.MaxInt - 2)
	assert.Equal(t, base.NonWrappingAdd(1), LineNumberFromZeroBased(math.MaxInt-1))
	assert.Equal(t, base.NonWrappingAdd(2), LineNumberFromZeroBased(math.MaxInt))
	assert.Equal(t, base.NonWrappingAdd(3), LineNumberFromZeroBased(math.MaxInt))
}

func TestNonWrappingAddNegative(t *testing.T) {
	base := LineNumberFromZeroBased(2)
	assert.Equal(t, base.NonWrappingAdd(-1), LineNumberFromZeroBased(1))
	assert.Equal(t, base.NonWrappingAdd(-2), LineNumberFromZeroBased(0))
	assert.Equal(t, base.NonWrappingAdd(-3), LineNumberFromZeroBased(0))
}

func TestLineNumberFormatting(t *testing.T) {
	assert.Equal(t, "1", LineNumberFromOneBased(1).Format())
	assert.Equal(t, "10", LineNumberFromOneBased(10).Format())
	assert.Equal(t, "100", LineNumberFromOneBased(100).Format())

	// Ref: // https://en.wikipedia.org/wiki/Decimal_separator#Exceptions_to_digit_grouping
	assert.Equal(t, "1000", LineNumberFromOneBased(1000).Format())

	assert.Equal(t, "10_000", LineNumberFromOneBased(10000).Format())
	assert.Equal(t, "100_000", LineNumberFromOneBased(100000).Format())
	assert.Equal(t, "1_000_000", LineNumberFromOneBased(1000000).Format())
	assert.Equal(t, "10_000_000", LineNumberFromOneBased(10000000).Format())
}

func TestLineNumberFromLength(t *testing.T) {
	// If the file has one line then the last zero based line number is 0.
	assert.Equal(t, LineNumberFromLength(1), LineNumberFromZeroBased(0))
}
