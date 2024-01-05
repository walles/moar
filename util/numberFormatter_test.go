package util

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestNumberFormatting(t *testing.T) {
	assert.Equal(t, "1", FormatNumber(1))
	assert.Equal(t, "10", FormatNumber(10))
	assert.Equal(t, "100", FormatNumber(100))

	// Ref: // https://en.wikipedia.org/wiki/Decimal_separator#Exceptions_to_digit_grouping
	assert.Equal(t, "1000", FormatNumber(1000))

	assert.Equal(t, "10_000", FormatNumber(10000))
	assert.Equal(t, "100_000", FormatNumber(100000))
	assert.Equal(t, "1_000_000", FormatNumber(1000000))
	assert.Equal(t, "10_000_000", FormatNumber(10000000))
}
