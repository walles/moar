package twin

import (
	"reflect"
	"testing"

	"gotest.tools/v3/assert"
)

func TestTrimSpaceRight(t *testing.T) {
	// Empty
	assert.Assert(t, reflect.DeepEqual(
		TrimSpaceRight(
			[]StyledRune{},
		),
		[]StyledRune{}))

	// Single non-space
	assert.Assert(t, reflect.DeepEqual(
		TrimSpaceRight(
			[]StyledRune{{Rune: 'x'}},
		),
		[]StyledRune{{Rune: 'x'}}))

	// Single space
	assert.Assert(t, reflect.DeepEqual(
		TrimSpaceRight(
			[]StyledRune{{Rune: ' '}},
		),
		[]StyledRune{}))

	// Non-space plus space
	assert.Assert(t, reflect.DeepEqual(
		TrimSpaceRight(
			[]StyledRune{{Rune: 'x'}, {Rune: ' '}},
		),
		[]StyledRune{{Rune: 'x'}}))
}
