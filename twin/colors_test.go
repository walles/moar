package twin

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestDownsample24BitsTo16Colors(t *testing.T) {
	assert.Equal(t,
		NewColor24Bit(255, 255, 255).downsampleTo(ColorCount16),
		NewColor16(15),
	)
}

func TestDownsample24BitsTo256Colors(t *testing.T) {
	assert.Equal(t,
		NewColor24Bit(255, 255, 255).downsampleTo(ColorCount256),

		// From https://jonasjacek.github.io/colors/
		NewColor256(231),
	)
}

func TestRealWorldDownsampling(t *testing.T) {
	assert.Equal(t,
		NewColor24Bit(0xd0, 0xd0, 0xd0).downsampleTo(ColorCount256),
		NewColor256(252), // From https://jonasjacek.github.io/colors/
	)
}

func TestAnsiStringWithDownSampling(t *testing.T) {
	assert.Equal(t,
		NewColor24Bit(0xd0, 0xd0, 0xd0).ansiString(true, ColorCount256),
		"\x1b[38;5;252m",
	)
}

func TestAnsiStringDefault(t *testing.T) {
	assert.Equal(t,
		ColorDefault.ansiString(true, ColorCount16),
		"\x1b[39m",
	)
}

func TestDistance(t *testing.T) {
	// Black -> white
	assert.Equal(t,
		NewColor24Bit(0, 0, 0).Distance(NewColor24Bit(255, 255, 255)),
		1.0,
	)

	// White -> black
	assert.Equal(t,
		NewColor24Bit(255, 255, 255).Distance(NewColor24Bit(0, 0, 0)),
		1.0,
	)
}
