package twin

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestColorRgbFirst16(t *testing.T) {
	r, g, b := color256ToRGB(5)

	assert.Equal(t, r, float64(0x80)/255.0)
	assert.Equal(t, g, float64(0x00)/255.0)
	assert.Equal(t, b, float64(0x80)/255.0)
}

func TestColorToRgbInTheGrey(t *testing.T) {
	r, g, b := color256ToRGB(252)

	assert.Equal(t, r, float64(0xd0)/255.0)
	assert.Equal(t, g, float64(0xd0)/255.0)
	assert.Equal(t, b, float64(0xd0)/255.0)
}

func TestColorToRgbInThe6x6Cube(t *testing.T) {
	r, g, b := color256ToRGB(101)

	assert.Equal(t, r, float64(0x87)/255.0)
	assert.Equal(t, g, float64(0x87)/255.0)
	assert.Equal(t, b, float64(0x5f)/255.0)
}

func TestColorToRgbStart6x6Cube(t *testing.T) {
	r, g, b := color256ToRGB(16)

	assert.Equal(t, r, float64(0x00)/255.0)
	assert.Equal(t, g, float64(0x00)/255.0)
	assert.Equal(t, b, float64(0x00)/255.0)
}

func TestColorRgbEnd6x6Cube(t *testing.T) {
	r, g, b := color256ToRGB(231)

	assert.Equal(t, r, float64(0xff)/255.0)
	assert.Equal(t, g, float64(0xff)/255.0)
	assert.Equal(t, b, float64(0xff)/255.0)
}
