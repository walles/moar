package main

import (
	"testing"

	"github.com/walles/moar/twin"
	"gotest.tools/v3/assert"
)

func TestParseScrollHint(t *testing.T) {
	token, err := parseScrollHint("ESC[7m>")
	assert.NilError(t, err)
	assert.Equal(t, token, twin.Cell{
		Rune:  '>',
		Style: twin.StyleDefault.WithAttr(twin.AttrReverse),
	})
}
