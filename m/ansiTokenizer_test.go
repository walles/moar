package m

import (
	"bufio"
	"log"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"gotest.tools/assert"

	"github.com/gdamore/tcell"
)

// Verify that we can tokenize all lines in ../sample-files/*
// without logging any errors
func TestTokenize(t *testing.T) {
	for _, fileName := range _GetTestFiles() {
		file, err := os.Open(fileName)
		if err != nil {
			t.Errorf("Error opening file <%s>: %s", fileName, err.Error())
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNumber := 0
		for scanner.Scan() {
			line := scanner.Text()
			lineNumber++

			var loglines strings.Builder
			logger := log.New(&loglines, "", 0)

			tokens, plainString := TokensFromString(logger, line)
			if len(tokens) != utf8.RuneCountInString(*plainString) {
				t.Errorf("%s:%d: len(tokens)=%d, len(plainString)=%d for: <%s>",
					fileName, lineNumber,
					len(tokens), utf8.RuneCountInString(*plainString), line)
				continue
			}

			if len(loglines.String()) != 0 {
				t.Errorf("%s: %s", fileName, loglines.String())
				continue
			}
		}
	}
}

func TestConsumeCompositeColorHappy(t *testing.T) {
	// 8 bit color
	// Example from: https://github.com/walles/moar/issues/14
	newIndex, color, err := consumeCompositeColor([]string{"38", "5", "74"}, 0, tcell.StyleDefault)
	assert.NilError(t, err)
	assert.Equal(t, newIndex, 3)
	assert.Equal(t, color, tcell.Color74)

	// 24 bit color
	newIndex, color, err = consumeCompositeColor([]string{"38", "2", "10", "20", "30"}, 0, tcell.StyleDefault)
	assert.NilError(t, err)
	assert.Equal(t, newIndex, 3)
	assert.Equal(t, color, tcell.NewRGBColor(10, 20, 30))
}

func TestConsumeCompositeColorHappyMidSequence(t *testing.T) {
	// 8 bit color
	// Example from: https://github.com/walles/moar/issues/14
	newIndex, color, err := consumeCompositeColor([]string{"whatever", "38", "5", "74"}, 1, tcell.StyleDefault)
	assert.NilError(t, err)
	assert.Equal(t, newIndex, 4)
	assert.Equal(t, color, tcell.Color74)

	// 24 bit color
	newIndex, color, err = consumeCompositeColor([]string{"whatever", "38", "2", "10", "20", "30"}, 1, tcell.StyleDefault)
	assert.NilError(t, err)
	assert.Equal(t, newIndex, 4)
	assert.Equal(t, color, tcell.NewRGBColor(10, 20, 30))
}

func TestConsumeCompositeColorBadPrefix(t *testing.T) {
	// 8 bit color
	// Example from: https://github.com/walles/moar/issues/14
	_, color, err := consumeCompositeColor([]string{"29"}, 0, tcell.StyleDefault)
	assert.Equal(t, err.Error, "Unknown start of color sequence <29>, expected 38 (foreground) or 48 (background): <CSI 29m>")
	assert.Equal(t, color, nil)

	// Same test but mid-sequence, with initial index > 0
	_, color, err = consumeCompositeColor([]string{"whatever", "29"}, 1, tcell.StyleDefault)
	assert.Equal(t, err.Error, "Unknown start of color sequence <29>, expected 38 (foreground) or 48 (background): <CSI 29m>")
	assert.Equal(t, color, nil)
}

func TestConsumeCompositeColorBadType(t *testing.T) {
	_, color, err := consumeCompositeColor([]string{"38", "4"}, 0, tcell.StyleDefault)
	// https://en.wikipedia.org/wiki/ANSI_escape_code#Colors
	assert.Equal(t, err.Error, "Unknown color type <4>, expected 5 (8 bit color) or 2 (24 bit color): <CSI 38;4m>")
	assert.Equal(t, color, nil)

	// Same test but mid-sequence, with initial index > 0
	_, color, err = consumeCompositeColor([]string{"whatever", "38", "4"}, 1, tcell.StyleDefault)
	assert.Equal(t, err.Error, "Unknown color type <4>, expected 5 (8 bit color) or 2 (24 bit color): <CSI 38;4m>")
	assert.Equal(t, color, nil)
}

func TestConsumeCompositeColorIncomplete8Bit(t *testing.T) {
	_, color, err := consumeCompositeColor([]string{"38", "5"}, 0, tcell.StyleDefault)
	assert.Equal(t, err.Error, "Incomplete 8 bit color sequence: <CSI 38;5m>")
	assert.Equal(t, color, nil)

	// Same test, mid-sequence
	_, color, err = consumeCompositeColor([]string{"whatever", "38", "5"}, 1, tcell.StyleDefault)
	assert.Equal(t, err.Error, "Incomplete 8 bit color sequence: <CSI 38;5m>")
	assert.Equal(t, color, nil)
}

// FIXME: Test incomplete 24 bit color sequence
