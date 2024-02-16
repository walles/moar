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

func TestPageOneInputFile(t *testing.T) {
	pager, screen, _, formatter, err := pagerFromArgs(
		[]string{"", "moar_test.go"},
		func(_ twin.MouseMode, _ twin.ColorType) (twin.Screen, error) {
			return twin.NewFakeScreen(80, 24), nil
		},
		false, // stdin is redirected
		false, // stdout is redirected
	)

	assert.NilError(t, err)
	assert.Assert(t, pager != nil)
	assert.Assert(t, screen != nil)
	assert.Assert(t, formatter != nil)
}
