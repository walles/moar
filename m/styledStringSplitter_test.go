package m

import (
	"testing"

	"github.com/walles/moar/twin"
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

// We should ignore OSC 133 sequences.
//
// Ref:
// https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md
func TestIgnorePromptHints(t *testing.T) {
	// From an e-mail I got titled "moar question: "--RAW-CONTROL-CHARS" equivalent"
	result := styledStringsFromString("\x1b]133;A\x1b\\hello")
	assert.Equal(t, twin.StyleDefault, result.trailer)
	assert.Equal(t, 1, len(result.styledStrings))
	assert.Equal(t, "hello", result.styledStrings[0].String)
	assert.Equal(t, twin.StyleDefault, result.styledStrings[0].Style)

	// C rather than A, different end-of-sequence, should also be ignored
	result = styledStringsFromString("\x1b]133;C\x07hello")
	assert.Equal(t, twin.StyleDefault, result.trailer)
	assert.Equal(t, 1, len(result.styledStrings))
	assert.Equal(t, "hello", result.styledStrings[0].String)
	assert.Equal(t, twin.StyleDefault, result.styledStrings[0].Style)
}
