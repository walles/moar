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

func TestRuneWidth(t *testing.T) {
	assert.Equal(t, NewStyledRune('x', Style{}).Width(), 1)
	assert.Equal(t, NewStyledRune('午', Style{}).Width(), 2)
}
