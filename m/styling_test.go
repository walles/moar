package m

import (
	"testing"

	"github.com/alecthomas/chroma/v2/styles"
	"gotest.tools/v3/assert"
)

func TestBackgroundStyleFromChromaGithub(t *testing.T) {
	style := backgroundStyleFromChroma(styles.Get("github"))
	assert.Check(t, style != nil)

	assert.Equal(t, style.String(), "Default color on #ffffff")
}

// FIXME: Add a parameterized test looping over all styles and checking that the
// contrast we get in backgroundStyleFromChroma is good enough.
