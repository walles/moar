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
