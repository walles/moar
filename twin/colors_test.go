package twin

import (
	"strings"
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
	actual := NewColor24Bit(0xd0, 0xd0, 0xd0).ansiString(colorTypeForeground, ColorCount256)
	actual = strings.ReplaceAll(actual, "\x1b", "ESC")
	expected := "ESC[38;5;252m"
	assert.Equal(t,
		actual,
		expected,
	)
}

func TestAnsiStringDefault(t *testing.T) {
	actual := ColorDefault.ansiString(colorTypeBackground, ColorCount16)
	actual = strings.ReplaceAll(actual, "\x1b", "ESC")
	expected := "ESC[49m"
	assert.Equal(t,
		actual,
		expected,
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
