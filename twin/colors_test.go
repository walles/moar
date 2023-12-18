package twin

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestDownsample24BitsTo16Colors(t *testing.T) {
	assert.Equal(t,
		NewColor24Bit(255, 255, 255).downsampleTo(ColorType16),
		NewColor16(15),
	)
}

func TestDownsample24BitsTo256Colors(t *testing.T) {
	assert.Equal(t,
		NewColor24Bit(255, 255, 255).downsampleTo(ColorType256),
		NewColor16(15),
	)
}

func TestRealWorldDownsampling(t *testing.T) {
	assert.Equal(t,
		NewColor24Bit(0xd0, 0xd0, 0xd0).downsampleTo(ColorType256),
		NewColor256(252), // From https://jonasjacek.github.io/colors/
	)
}

func TestAnsiStringWithDownSampling(t *testing.T) {
	assert.Equal(t,
		NewColor24Bit(0xd0, 0xd0, 0xd0).ansiString(true, ColorType256),
		"\x1b[38;5;252m",
	)
}
