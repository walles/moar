package m

import (
	"testing"

	"github.com/alecthomas/chroma/v2/styles"
	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

func TestBackgroundStyleFromChromaGithub(t *testing.T) {
	style := backgroundStyleFromChroma(*styles.Get("github"))

	assert.Equal(t, style.String(), "Default color on #ffffff")
}

func TestBackgroundStyleFromChromaAverage(t *testing.T) {
	// Verify we get the right values out of a style with both Text and Background.
	style := backgroundStyleFromChroma(*styles.Get("average"))
	assert.Equal(t, style.String(), "#757575 on #000000")
}

func TestBackgroundStyleFromChromaNative(t *testing.T) {
	// Verify we get the right values out of a style where Background comes with
	// both foreground and background.
	style := backgroundStyleFromChroma(*styles.Get("native"))
	assert.Equal(t, style.String(), "#d0d0d0 on #202020")
}

// Loop over all styles and check that the contrast we get in
// backgroundStyleFromChroma is good enough.
func TestBackgroundStyleContrast(t *testing.T) {
	for _, style := range styles.Registry {
		t.Run(style.Name, func(t *testing.T) {
			backgroundStyle := backgroundStyleFromChroma(*style)

			if backgroundStyle.Foreground().ColorType() == twin.ColorTypeDefault {
				return
			}

			if backgroundStyle.Background().ColorType() == twin.ColorTypeDefault {
				return
			}

			distance := backgroundStyle.Background().Distance(backgroundStyle.Foreground())

			// 0.4 feels low, but as of Chroma v2.12.0 this is the distance we
			// get for some styles (solarized-dark, solarized-light, average).
			//
			// Let's at least verify it doesn't get worse than this.
			assert.Check(t, distance > 0.4, "distance=%f", distance)
		})
	}
}
