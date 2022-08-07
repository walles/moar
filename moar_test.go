package main

import (
	"testing"

	"github.com/walles/moar/twin"
	"gotest.tools/assert"
)

func TestParseScrollHint(t *testing.T) {
	token := parseScrollHint("ESC[7m>", nil)
	assert.Equal(t, token, twin.Cell{
		Rune:  '>',
		Style: twin.StyleDefault.WithAttr(twin.AttrReverse),
	})
}
