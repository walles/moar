package util

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestFormatInt(t *testing.T) {
	assert.Equal(t, "1", FormatInt(1))
	assert.Equal(t, "10", FormatInt(10))
	assert.Equal(t, "100", FormatInt(100))

	// Ref: // https://en.wikipedia.org/wiki/Decimal_separator#Exceptions_to_digit_grouping
	assert.Equal(t, "1000", FormatInt(1000))

	assert.Equal(t, "10_000", FormatInt(10000))
	assert.Equal(t, "100_000", FormatInt(100000))
	assert.Equal(t, "1_000_000", FormatInt(1000000))
	assert.Equal(t, "10_000_000", FormatInt(10000000))
}
