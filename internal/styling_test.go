package internal

import (
	"os"
	"testing"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/walles/moor/v2/twin"
	"gotest.tools/v3/assert"
)

func TestTwinStyleFromChroma(t *testing.T) {
	// Test that getting exact GenericHeading from base16-snazzy works
	style := twinStyleFromChroma(
		styles.Registry["base16-snazzy"],
		&formatters.TTY16m,
		chroma.GenericHeading,
		true,
	)

	assert.Equal(t,
		*style,
		twin.StyleDefault.
			WithAttr(twin.AttrBold).
			WithForeground(twin.NewColor24Bit(0xe2, 0xe4, 0xe5)))
}

func TestSetStyle(t *testing.T) {
	assert.NilError(t, os.Setenv("MOOR_TEST_STYLE", "\x1b[1;31m"))
	style := twin.StyleDefault
	setStyle(&style, "MOOR_TEST_STYLE", nil)

	assert.Equal(t, style, twin.StyleDefault.WithAttr(twin.AttrBold).WithForeground(twin.NewColor16(1)))
}
