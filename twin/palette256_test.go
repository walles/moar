package twin

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestColorRgbFirst16(t *testing.T) {
	r, g, b := color256ToRGB(5)

	assert.Equal(t, r, uint8(0x80))
	assert.Equal(t, g, uint8(0x00))
	assert.Equal(t, b, uint8(0x80))
}

func TestColorToRgbInTheGrey(t *testing.T) {
	r, g, b := color256ToRGB(252)

	assert.Equal(t, r, uint8(0xd0))
	assert.Equal(t, g, uint8(0xd0))
	assert.Equal(t, b, uint8(0xd0))
}

func TestColorToRgbInThe6x6Cube(t *testing.T) {
	r, g, b := color256ToRGB(101)

	assert.Equal(t, r, uint8(0x87))
	assert.Equal(t, g, uint8(0x87))
	assert.Equal(t, b, uint8(0x5f))
}

func TestColorToRgbStart6x6Cube(t *testing.T) {
	r, g, b := color256ToRGB(16)

	assert.Equal(t, r, uint8(0x00))
	assert.Equal(t, g, uint8(0x00))
	assert.Equal(t, b, uint8(0x00))
}

func TestColorRgbEnd6x6Cube(t *testing.T) {
	r, g, b := color256ToRGB(231)

	assert.Equal(t, r, uint8(0xff))
	assert.Equal(t, g, uint8(0xff))
	assert.Equal(t, b, uint8(0xff))
}
