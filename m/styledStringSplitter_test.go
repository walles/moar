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

func collectStyledStrings(s string) ([]_StyledString, twin.Style) {
	styledStrings := []_StyledString{}
	trailer := styledStringsFromString(s, nil, func(str string, style twin.Style) {
		styledStrings = append(styledStrings, _StyledString{str, style})
	})
	return styledStrings, trailer
}

// We should ignore OSC 133 sequences.
//
// Ref:
// https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md
func TestIgnorePromptHints(t *testing.T) {
	// From an e-mail I got titled "moar question: "--RAW-CONTROL-CHARS" equivalent"
	styledStrings, trailer := collectStyledStrings("\x1b]133;A\x1b\\hello")
	assert.Equal(t, twin.StyleDefault, trailer)
	assert.Equal(t, 1, len(styledStrings))
	assert.Equal(t, "hello", styledStrings[0].String)
	assert.Equal(t, twin.StyleDefault, styledStrings[0].Style)

	// C rather than A, different end-of-sequence, should also be ignored
	styledStrings, trailer = collectStyledStrings("\x1b]133;C\x07hello")
	assert.Equal(t, twin.StyleDefault, trailer)
	assert.Equal(t, 1, len(styledStrings))
	assert.Equal(t, "hello", styledStrings[0].String)
	assert.Equal(t, twin.StyleDefault, styledStrings[0].Style)
}

// Unsure why colon separated colors exist, but the fact is that a number of
// things emit colon separated SGR codes. And numerous terminals accept them
// (search page for "delimiter"): https://github.com/kovidgoyal/kitty/issues/7
//
// Johan got an e-mail titled "moar question: "--RAW-CONTROL-CHARS" equivalent"
// about the sequence we're testing here.
func TestColonColors(t *testing.T) {
	styledStrings, trailer := collectStyledStrings("\x1b[38:5:238mhello")
	assert.Equal(t, twin.StyleDefault, trailer)
	assert.Equal(t, 1, len(styledStrings))
	assert.Equal(t, "hello", styledStrings[0].String)
	assert.Equal(t, twin.StyleDefault.Foreground(twin.NewColor256(238)), styledStrings[0].Style)
}
