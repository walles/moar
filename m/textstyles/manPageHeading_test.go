package textstyles

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestIsManPageHeading(t *testing.T) {
	assert.Assert(t, !isManPageHeading(""))
	assert.Assert(t, !isManPageHeading("A"), "Incomplete sequence")
	assert.Assert(t, !isManPageHeading("A\b"), "Incomplete sequence")

	assert.Assert(t, isManPageHeading("A\bA"))
	assert.Assert(t, isManPageHeading("A\bA B\bB"), "Whitespace can be not-bold")

	assert.Assert(t, !isManPageHeading("A\bC"), "Different first and last char")
	assert.Assert(t, !isManPageHeading("a\ba"), "Not ALL CAPS")

	assert.Assert(t, !isManPageHeading("A\bAX"), "Incomplete sequence")

	assert.Assert(t, !isManPageHeading(" \b "), "Headings do not start with space")
}
