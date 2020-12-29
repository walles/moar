package m

import (
	"bufio"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	log "github.com/sirupsen/logrus"

	"gotest.tools/assert"

	"github.com/gdamore/tcell/v2"
)

// Verify that we can tokenize all lines in ../sample-files/*
// without logging any errors
func TestTokenize(t *testing.T) {
	for _, fileName := range getTestFiles() {
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
			log.SetOutput(&loglines)

			tokens, plainString := TokensFromString(line)
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

func TestUnderline(t *testing.T) {
	tokens, _ := TokensFromString("a\x1b[4mb\x1b[24mc")
	assert.Equal(t, len(tokens), 3)
	assert.Equal(t, tokens[0], Token{Rune: 'a', Style: tcell.StyleDefault})
	assert.Equal(t, tokens[1], Token{Rune: 'b', Style: tcell.StyleDefault.Underline(true)})
	assert.Equal(t, tokens[2], Token{Rune: 'c', Style: tcell.StyleDefault})
}

func TestManPages(t *testing.T) {
	// Bold
	tokens, _ := TokensFromString("ab\bbc")
	assert.Equal(t, len(tokens), 3)
	assert.Equal(t, tokens[0], Token{Rune: 'a', Style: tcell.StyleDefault})
	assert.Equal(t, tokens[1], Token{Rune: 'b', Style: tcell.StyleDefault.Bold(true)})
	assert.Equal(t, tokens[2], Token{Rune: 'c', Style: tcell.StyleDefault})

	// Underline
	tokens, _ = TokensFromString("a_\bbc")
	assert.Equal(t, len(tokens), 3)
	assert.Equal(t, tokens[0], Token{Rune: 'a', Style: tcell.StyleDefault})
	assert.Equal(t, tokens[1], Token{Rune: 'b', Style: tcell.StyleDefault.Underline(true)})
	assert.Equal(t, tokens[2], Token{Rune: 'c', Style: tcell.StyleDefault})

	// Bullet point 1, taken from doing this on my macOS system:
	// env PAGER="hexdump -C" man printf | moar
	tokens, _ = TokensFromString("a+\b+\bo\bob")
	assert.Equal(t, len(tokens), 3)
	assert.Equal(t, tokens[0], Token{Rune: 'a', Style: tcell.StyleDefault})
	assert.Equal(t, tokens[1], Token{Rune: '•', Style: tcell.StyleDefault})
	assert.Equal(t, tokens[2], Token{Rune: 'b', Style: tcell.StyleDefault})

	// Bullet point 2, taken from doing this using the "fish" shell on my macOS system:
	// man printf | hexdump -C | moar
	tokens, _ = TokensFromString("a+\bob")
	assert.Equal(t, len(tokens), 3)
	assert.Equal(t, tokens[0], Token{Rune: 'a', Style: tcell.StyleDefault})
	assert.Equal(t, tokens[1], Token{Rune: '•', Style: tcell.StyleDefault})
	assert.Equal(t, tokens[2], Token{Rune: 'b', Style: tcell.StyleDefault})
}

func TestConsumeCompositeColorHappy(t *testing.T) {
	// 8 bit color
	// Example from: https://github.com/walles/moar/issues/14
	newIndex, color, err := consumeCompositeColor([]string{"38", "5", "74"}, 0)
	assert.NilError(t, err)
	assert.Equal(t, newIndex, 3)
	assert.Equal(t, *color, tcell.Color74)

	// 24 bit color
	newIndex, color, err = consumeCompositeColor([]string{"38", "2", "10", "20", "30"}, 0)
	assert.NilError(t, err)
	assert.Equal(t, newIndex, 5)
	assert.Equal(t, *color, tcell.NewRGBColor(10, 20, 30))
}

func TestConsumeCompositeColorHappyMidSequence(t *testing.T) {
	// 8 bit color
	// Example from: https://github.com/walles/moar/issues/14
	newIndex, color, err := consumeCompositeColor([]string{"whatever", "38", "5", "74"}, 1)
	assert.NilError(t, err)
	assert.Equal(t, newIndex, 4)
	assert.Equal(t, *color, tcell.Color74)

	// 24 bit color
	newIndex, color, err = consumeCompositeColor([]string{"whatever", "38", "2", "10", "20", "30"}, 1)
	assert.NilError(t, err)
	assert.Equal(t, newIndex, 6)
	assert.Equal(t, *color, tcell.NewRGBColor(10, 20, 30))
}

func TestConsumeCompositeColorBadPrefix(t *testing.T) {
	// 8 bit color
	// Example from: https://github.com/walles/moar/issues/14
	_, color, err := consumeCompositeColor([]string{"29"}, 0)
	assert.Equal(t, err.Error(), "Unknown start of color sequence <29>, expected 38 (foreground) or 48 (background): <CSI 29m>")
	assert.Assert(t, color == nil)

	// Same test but mid-sequence, with initial index > 0
	_, color, err = consumeCompositeColor([]string{"whatever", "29"}, 1)
	assert.Equal(t, err.Error(), "Unknown start of color sequence <29>, expected 38 (foreground) or 48 (background): <CSI 29m>")
	assert.Assert(t, color == nil)
}

func TestConsumeCompositeColorBadType(t *testing.T) {
	_, color, err := consumeCompositeColor([]string{"38", "4"}, 0)
	// https://en.wikipedia.org/wiki/ANSI_escape_code#Colors
	assert.Equal(t, err.Error(), "Unknown color type <4>, expected 5 (8 bit color) or 2 (24 bit color): <CSI 38;4m>")
	assert.Assert(t, color == nil)

	// Same test but mid-sequence, with initial index > 0
	_, color, err = consumeCompositeColor([]string{"whatever", "38", "4"}, 1)
	assert.Equal(t, err.Error(), "Unknown color type <4>, expected 5 (8 bit color) or 2 (24 bit color): <CSI 38;4m>")
	assert.Assert(t, color == nil)
}

func TestConsumeCompositeColorIncomplete(t *testing.T) {
	_, color, err := consumeCompositeColor([]string{"38"}, 0)
	assert.Equal(t, err.Error(), "Incomplete color sequence: <CSI 38m>")
	assert.Assert(t, color == nil)

	// Same test, mid-sequence
	_, color, err = consumeCompositeColor([]string{"whatever", "38"}, 1)
	assert.Equal(t, err.Error(), "Incomplete color sequence: <CSI 38m>")
	assert.Assert(t, color == nil)
}

func TestConsumeCompositeColorIncomplete8Bit(t *testing.T) {
	_, color, err := consumeCompositeColor([]string{"38", "5"}, 0)
	assert.Equal(t, err.Error(), "Incomplete 8 bit color sequence: <CSI 38;5m>")
	assert.Assert(t, color == nil)

	// Same test, mid-sequence
	_, color, err = consumeCompositeColor([]string{"whatever", "38", "5"}, 1)
	assert.Equal(t, err.Error(), "Incomplete 8 bit color sequence: <CSI 38;5m>")
	assert.Assert(t, color == nil)
}

func TestConsumeCompositeColorIncomplete24Bit(t *testing.T) {
	_, color, err := consumeCompositeColor([]string{"38", "2", "10", "20"}, 0)
	assert.Equal(t, err.Error(), "Incomplete 24 bit color sequence, expected N8;2;R;G;Bm: <CSI 38;2;10;20m>")
	assert.Assert(t, color == nil)

	// Same test, mid-sequence
	_, color, err = consumeCompositeColor([]string{"whatever", "38", "2", "10", "20"}, 1)
	assert.Equal(t, err.Error(), "Incomplete 24 bit color sequence, expected N8;2;R;G;Bm: <CSI 38;2;10;20m>")
	assert.Assert(t, color == nil)
}

func TestUpdateStyle(t *testing.T) {
	numberColored := updateStyle(tcell.StyleDefault, "\x1b[33m")
	assert.Equal(t, numberColored, tcell.StyleDefault.Foreground(tcell.ColorOlive))
}
