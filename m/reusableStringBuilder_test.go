package m

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestReusableStringBuilder_Basics(t *testing.T) {
	testMe := reusableStringBuilder{}
	assert.Equal(t, "", testMe.String())

	testMe.WriteRune('ä')
	assert.Equal(t, "ä", testMe.String())

	testMe.Reset()
	assert.Equal(t, "", testMe.String())
}

// Ensure the strings returned by our builder are real, rather than just being
// views into its backing array.
func TestReusableStringBuilder_Copies(t *testing.T) {
	testMe := reusableStringBuilder{}

	testMe.WriteRune('a')
	s1 := testMe.String()

	testMe.Reset()
	testMe.WriteRune('b')
	s2 := testMe.String()

	assert.Equal(t, s1, "a")
	assert.Equal(t, s2, "b")
}
