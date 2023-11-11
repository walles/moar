package m

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestNextCharLastChar_base(t *testing.T) {
	s := styledStringSplitter{
		input: "a",
	}

	assert.Equal(t, 'a', s.nextChar())
	assert.Equal(t, 'a', s.lastChar())
	assert.Equal(t, rune(-1), s.nextChar())
	assert.Equal(t, rune(-1), s.lastChar())
}

func TestNextCharLastChar_empty(t *testing.T) {
	s := styledStringSplitter{
		input: "",
	}

	assert.Equal(t, rune(-1), s.nextChar())
	assert.Equal(t, rune(-1), s.lastChar())
}
