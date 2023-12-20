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

// Test looping over all styles and checking that the contrast we get in
// backgroundStyleFromChroma is good enough.
func TestBackgroundStyleContrast(t *testing.T) {
	for _, style := range styles.Registry {
		t.Run(style.Name, func(t *testing.T) {
			backgroundStyle := backgroundStyleFromChroma(*style)
			assert.Check(t, backgroundStyle.Background().ColorType() == twin.ColorType24bit)

			if backgroundStyle.Foreground().ColorType() == twin.ColorTypeDefault {
				return
			}

			distance := backgroundStyle.Background().Distance(backgroundStyle.Foreground())
			assert.Check(t, distance > 0.5, "distance=%f", distance)
		})
	}
}
