package twin

import (
	"testing"

	"gotest.tools/assert"
)

func TestLuminance(t *testing.T) {
	// Black luminance should be 0
	if luminance, err := NewColor24Bit(0, 0, 0).Luminance(); err != nil {
		panic(err)
	} else {
		assert.Equal(t, luminance, 0)
	}

	// White luminance should be 255
	if luminance, err := NewColor24Bit(255, 255, 255).Luminance(); err != nil {
		panic(err)
	} else {
		assert.Equal(t, luminance, 255)
	}

	blue_luminance, err := NewColor24Bit(0, 0, 255).Luminance()
	if err != nil {
		panic(err)
	}

	red_luminance, err := NewColor24Bit(255, 0, 0).Luminance()
	if err != nil {
		panic(err)
	}

	green_luminance, err := NewColor24Bit(0, 255, 0).Luminance()
	if err != nil {
		panic(err)
	}

	assert.Assert(t, blue_luminance > 0)
	assert.Assert(t, red_luminance > blue_luminance)
	assert.Assert(t, green_luminance > red_luminance)
	assert.Assert(t, green_luminance < 255)
}
