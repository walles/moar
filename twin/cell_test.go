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
			[]Cell{},
		),
		[]Cell{}))

	// Single non-space
	assert.Assert(t, reflect.DeepEqual(
		TrimSpaceRight(
			[]Cell{{Rune: 'x'}},
		),
		[]Cell{{Rune: 'x'}}))

	// Single space
	assert.Assert(t, reflect.DeepEqual(
		TrimSpaceRight(
			[]Cell{{Rune: ' '}},
		),
		[]Cell{}))

	// Non-space plus space
	assert.Assert(t, reflect.DeepEqual(
		TrimSpaceRight(
			[]Cell{{Rune: 'x'}, {Rune: ' '}},
		),
		[]Cell{{Rune: 'x'}}))
}
