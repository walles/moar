package m

import (
	"testing"

	"gotest.tools/assert"
)

func TestNumberFormatting(t *testing.T) {
	assert.Equal(t, "1", formatNumber(1))
	assert.Equal(t, "10", formatNumber(10))
	assert.Equal(t, "100", formatNumber(100))
	assert.Equal(t, "1_000", formatNumber(1000))
	assert.Equal(t, "10_000", formatNumber(10000))
	assert.Equal(t, "100_000", formatNumber(100000))
	assert.Equal(t, "1_000_000", formatNumber(1000000))
	assert.Equal(t, "10_000_000", formatNumber(10000000))
}
