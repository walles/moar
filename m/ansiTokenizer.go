package m

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/twin"
)

const _TabSize = 4

const BACKSPACE = '\b'

var manPageBold = twin.StyleDefault.WithAttr(twin.AttrBold)
var manPageUnderline = twin.StyleDefault.WithAttr(twin.AttrUnderline)

// ESC[...m: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR
var sgrSequencePattern = regexp.MustCompile("\x1b\\[([0-9;]*m)")

// A Line represents a line of text that can / will be paged
type Line struct {
	raw   string
	plain *string
}

// NewLine creates a new Line from a (potentially ANSI / man page formatted) string
func NewLine(raw string) *Line {
	return &Line{
		raw:   raw,
		plain: nil,
	}
}

// Tokens returns a representation of the string split into styled tokens
func (line *Line) Tokens() []twin.Cell {
	return cellsFromString(line.raw)
}

// Plain returns a plain text representation of the initial string
func (line *Line) Plain() string {
	if line.plain == nil {
		plain := withoutFormatting(line.raw)
		line.plain = &plain
	}
	return *line.plain
}

// SetManPageFormatFromEnv parses LESS_TERMCAP_xx environment variables and
// adapts the moar output accordingly.
func SetManPageFormatFromEnv() {
	// Requested here: https://github.com/walles/moar/issues/14

	lessTermcapMd := os.Getenv("LESS_TERMCAP_md")
	if lessTermcapMd != "" {
		manPageBold = termcapToStyle(lessTermcapMd)
	}

	lessTermcapUs := os.Getenv("LESS_TERMCAP_us")
	if lessTermcapUs != "" {
		manPageUnderline = termcapToStyle(lessTermcapUs)
	}
}

func termcapToStyle(termcap string) twin.Style {
	// Add a character to be sure we have one to take the format from
	cells := cellsFromString(termcap + "x")
	return cells[len(cells)-1].Style
}

func isPlain(s string) bool {
	for i := 0; i < len(s); i++ {
		byteAtIndex := s[i]
		if byteAtIndex < 32 {
			return false
		}
		if byteAtIndex > 127 {
			return false
		}
	}

	return true
}

func withoutFormatting(s string) string {
	if isPlain(s) {
		return s
	}

	stripped := strings.Builder{}
	runeCount := 0
	stripped.Grow(len(s))
	for _, styledString := range styledStringsFromString(s) {
		for _, runeValue := range runesFromStyledString(styledString) {
			switch runeValue {

			case '\x09': // TAB
				for {
					stripped.WriteRune(' ')
					runeCount++

					if runeCount%_TabSize == 0 {
						// We arrived at the next tab stop
						break
					}
				}

			case '�': // Go's broken-UTF8 marker
				stripped.WriteRune('?')
				runeCount++

			case BACKSPACE:
				stripped.WriteRune('<')
				runeCount++

			default:
				if !unicode.IsGraphic(runeValue) {
					stripped.WriteRune('?')
					runeCount++
					continue
				}
				stripped.WriteRune(runeValue)
				runeCount++
			}
		}
	}

	return stripped.String()
}

// Turn a (formatted) string into a series of screen cells
func cellsFromString(s string) []twin.Cell {
	var cells []twin.Cell

	// Specs: https://en.wikipedia.org/wiki/ANSI_escape_code#3-bit_and_4-bit
	styleBrokenUtf8 := twin.StyleDefault.Background(twin.NewColor16(1)).Foreground(twin.NewColor16(7))

	for _, styledString := range styledStringsFromString(s) {
		for _, token := range tokensFromStyledString(styledString) {
			switch token.Rune {

			case '\x09': // TAB
				for {
					cells = append(cells, twin.Cell{
						Rune:  ' ',
						Style: styledString.Style,
					})

					if (len(cells))%_TabSize == 0 {
						// We arrived at the next tab stop
						break
					}
				}

			case '�': // Go's broken-UTF8 marker
				cells = append(cells, twin.Cell{
					Rune:  '?',
					Style: styleBrokenUtf8,
				})

			case BACKSPACE:
				cells = append(cells, twin.Cell{
					Rune:  '<',
					Style: styleBrokenUtf8,
				})

			default:
				if !unicode.IsGraphic(token.Rune) {
					cells = append(cells, twin.Cell{
						Rune:  '?',
						Style: styleBrokenUtf8,
					})
					continue
				}
				cells = append(cells, token)
			}
		}
	}

	return cells
}

// Consume 'x<x', where '<' is backspace and the result is a bold 'x'
func consumeBold(runes []rune, index int) (int, *twin.Cell) {
	if index+2 >= len(runes) {
		// Not enough runes left for a bold
		return index, nil
	}

	if runes[index+1] != BACKSPACE {
		// No backspace in the middle, never mind
		return index, nil
	}

	if runes[index] != runes[index+2] {
		// First and last rune not the same, never mind
		return index, nil
	}

	// We have a match!
	return index + 3, &twin.Cell{
		Rune:  runes[index],
		Style: manPageBold,
	}
}

// Consume '_<x', where '<' is backspace and the result is an underlined 'x'
func consumeUnderline(runes []rune, index int) (int, *twin.Cell) {
	if index+2 >= len(runes) {
		// Not enough runes left for a underline
		return index, nil
	}

	if runes[index+1] != BACKSPACE {
		// No backspace in the middle, never mind
		return index, nil
	}

	if runes[index] != '_' {
		// No underline, never mind
		return index, nil
	}

	// We have a match!
	return index + 3, &twin.Cell{
		Rune:  runes[index+2],
		Style: manPageUnderline,
	}
}

// Consume '+<+<o<o' / '+<o', where '<' is backspace and the result is a unicode bullet.
//
// Used on man pages, try "man printf" on macOS for one example.
func consumeBullet(runes []rune, index int) (int, *twin.Cell) {
	patterns := [][]byte{[]byte("+\bo"), []byte("+\b+\bo\bo")}
	for _, pattern := range patterns {
		if index+len(pattern) > len(runes) {
			// Not enough runes left for a bullet
			continue
		}

		mismatch := false
		for delta, patternByte := range pattern {
			if patternByte != byte(runes[index+delta]) {
				// Bullet pattern mismatch, never mind
				mismatch = true
				break
			}
		}
		if mismatch {
			continue
		}

		// We have a match!
		return index + len(pattern), &twin.Cell{
			Rune:  '•', // Unicode bullet point
			Style: twin.StyleDefault,
		}
	}

	return index, nil
}

func runesFromStyledString(styledString _StyledString) string {
	hasBackspace := false
	for _, byteValue := range []byte(styledString.String) {
		if byteValue == BACKSPACE {
			hasBackspace = true
			break
		}
	}

	if !hasBackspace {
		// Shortcut when there's no backspace based formatting to worry about
		return styledString.String
	}

	// Special handling for man page formatted lines
	cells := tokensFromStyledString(styledString)
	returnMe := strings.Builder{}
	returnMe.Grow(len(cells))
	for _, cell := range cells {
		returnMe.WriteRune(cell.Rune)
	}

	return returnMe.String()
}

func tokensFromStyledString(styledString _StyledString) []twin.Cell {
	runes := []rune(styledString.String)

	hasBackspace := false
	for _, runeValue := range runes {
		if runeValue == BACKSPACE {
			hasBackspace = true
			break
		}
	}

	tokens := make([]twin.Cell, 0, len(runes))
	if !hasBackspace {
		// Shortcut when there's no backspace based formatting to worry about
		for _, runeValue := range runes {
			tokens = append(tokens, twin.Cell{
				Rune:  runeValue,
				Style: styledString.Style,
			})
		}
		return tokens
	}

	// Special handling for man page formatted lines
	for index := 0; index < len(runes); index++ {
		nextIndex, token := consumeBullet(runes, index)
		if nextIndex != index {
			tokens = append(tokens, *token)
			index = nextIndex - 1
			continue
		}

		nextIndex, token = consumeBold(runes, index)
		if nextIndex != index {
			tokens = append(tokens, *token)
			index = nextIndex - 1
			continue
		}

		nextIndex, token = consumeUnderline(runes, index)
		if nextIndex != index {
			tokens = append(tokens, *token)
			index = nextIndex - 1
			continue
		}

		tokens = append(tokens, twin.Cell{
			Rune:  runes[index],
			Style: styledString.Style,
		})
	}

	return tokens
}

type _StyledString struct {
	String string
	Style  twin.Style
}

func styledStringsFromString(s string) []_StyledString {
	hasEsc := false
	for _, currentByte := range []byte(s) {
		if currentByte == '\x1b' {
			hasEsc = true
			break
		}
	}
	if !hasEsc {
		return []_StyledString{{
			String: s,
			Style:  twin.StyleDefault,
		}}
	}

	// The rest of this function was inspired by the
	// https://golang.org/pkg/regexp/#Regexp.Split source code

	matches := sgrSequencePattern.FindAllStringIndex(s, -1)
	styledStrings := make([]_StyledString, 0, len(matches)+1)

	style := twin.StyleDefault

	beg := 0
	end := 0
	for _, match := range matches {
		end = match[0]

		if end > beg {
			// Found non-zero length string
			styledStrings = append(styledStrings, _StyledString{
				String: s[beg:end],
				Style:  style,
			})
		}

		matchedPart := s[match[0]:match[1]]
		style = updateStyle(style, matchedPart)

		beg = match[1]
	}

	if end != len(s) {
		styledStrings = append(styledStrings, _StyledString{
			String: s[beg:],
			Style:  style,
		})
	}

	return styledStrings
}

// updateStyle parses a string of the form "ESC[33m" into changes to style
func updateStyle(style twin.Style, escapeSequence string) twin.Style {
	numbers := strings.Split(escapeSequence[2:len(escapeSequence)-1], ";")
	index := 0
	for index < len(numbers) {
		number := numbers[index]
		index++
		switch strings.TrimLeft(number, "0") {
		case "":
			style = twin.StyleDefault

		case "1":
			style = style.WithAttr(twin.AttrBold)

		case "2":
			style = style.WithAttr(twin.AttrDim)

		case "3":
			style = style.WithAttr(twin.AttrItalic)

		case "4":
			style = style.WithAttr(twin.AttrUnderline)

		case "7":
			style = style.WithAttr(twin.AttrReverse)

		case "22":
			style = style.WithoutAttr(twin.AttrBold).WithoutAttr(twin.AttrDim)

		case "23":
			style = style.WithoutAttr(twin.AttrItalic)

		case "24":
			style = style.WithoutAttr(twin.AttrUnderline)

		case "27":
			style = style.WithoutAttr(twin.AttrReverse)

		// Foreground colors, https://pkg.go.dev/github.com/gdamore/tcell#Color
		case "30":
			style = style.Foreground(twin.NewColor16(0))
		case "31":
			style = style.Foreground(twin.NewColor16(1))
		case "32":
			style = style.Foreground(twin.NewColor16(2))
		case "33":
			style = style.Foreground(twin.NewColor16(3))
		case "34":
			style = style.Foreground(twin.NewColor16(4))
		case "35":
			style = style.Foreground(twin.NewColor16(5))
		case "36":
			style = style.Foreground(twin.NewColor16(6))
		case "37":
			style = style.Foreground(twin.NewColor16(7))
		case "38":
			var err error = nil
			var color *twin.Color
			index, color, err = consumeCompositeColor(numbers, index-1)
			if err != nil {
				log.Warnf("Foreground: %s", err.Error())
				return style
			}
			style = style.Foreground(*color)
		case "39":
			style = style.Foreground(twin.ColorDefault)

		// Background colors, see https://pkg.go.dev/github.com/gdamore/Color
		case "40":
			style = style.Background(twin.NewColor16(0))
		case "41":
			style = style.Background(twin.NewColor16(1))
		case "42":
			style = style.Background(twin.NewColor16(2))
		case "43":
			style = style.Background(twin.NewColor16(3))
		case "44":
			style = style.Background(twin.NewColor16(4))
		case "45":
			style = style.Background(twin.NewColor16(5))
		case "46":
			style = style.Background(twin.NewColor16(6))
		case "47":
			style = style.Background(twin.NewColor16(7))
		case "48":
			var err error = nil
			var color *twin.Color
			index, color, err = consumeCompositeColor(numbers, index-1)
			if err != nil {
				log.Warnf("Background: %s", err.Error())
				return style
			}
			style = style.Background(*color)
		case "49":
			style = style.Background(twin.ColorDefault)

		// Bright foreground colors: see https://pkg.go.dev/github.com/gdamore/Color
		//
		// After testing vs less and cat on iTerm2 3.3.9 / macOS Catalina
		// 10.15.4 that's how they seem to handle this, tested with:
		// * TERM=xterm-256color
		// * TERM=screen-256color
		case "90":
			style = style.Foreground(twin.NewColor16(8))
		case "91":
			style = style.Foreground(twin.NewColor16(9))
		case "92":
			style = style.Foreground(twin.NewColor16(10))
		case "93":
			style = style.Foreground(twin.NewColor16(11))
		case "94":
			style = style.Foreground(twin.NewColor16(12))
		case "95":
			style = style.Foreground(twin.NewColor16(13))
		case "96":
			style = style.Foreground(twin.NewColor16(14))
		case "97":
			style = style.Foreground(twin.NewColor16(15))

		default:
			log.Warnf("Unrecognized ANSI SGR code <%s>", number)
		}
	}

	return style
}

// numbers is a list of numbers from a ANSI SGR string
// index points to either 38 or 48 in that string
//
// This method will return:
// * The first index in the string that this function did not consume
// * A color value that can be applied to a style
func consumeCompositeColor(numbers []string, index int) (int, *twin.Color, error) {
	baseIndex := index
	if numbers[index] != "38" && numbers[index] != "48" {
		err := fmt.Errorf(
			"unknown start of color sequence <%s>, expected 38 (foreground) or 48 (background): <CSI %sm>",
			numbers[index],
			strings.Join(numbers[baseIndex:], ";"))
		return -1, nil, err
	}

	index++
	if index >= len(numbers) {
		err := fmt.Errorf(
			"incomplete color sequence: <CSI %sm>",
			strings.Join(numbers[baseIndex:], ";"))
		return -1, nil, err
	}

	if numbers[index] == "5" {
		// Handle 8 bit color
		index++
		if index >= len(numbers) {
			err := fmt.Errorf(
				"incomplete 8 bit color sequence: <CSI %sm>",
				strings.Join(numbers[baseIndex:], ";"))
			return -1, nil, err
		}

		colorNumber, err := strconv.Atoi(numbers[index])
		if err != nil {
			return -1, nil, err
		}

		colorValue := twin.NewColor256(uint8(colorNumber))
		return index + 1, &colorValue, nil
	}

	if numbers[index] == "2" {
		// Handle 24 bit color
		rIndex := index + 1
		gIndex := index + 2
		bIndex := index + 3
		if bIndex >= len(numbers) {
			err := fmt.Errorf(
				"incomplete 24 bit color sequence, expected N8;2;R;G;Bm: <CSI %sm>",
				strings.Join(numbers[baseIndex:], ";"))
			return -1, nil, err
		}

		rValueX, err := strconv.ParseInt(numbers[rIndex], 10, 32)
		if err != nil {
			return -1, nil, err
		}
		rValue := uint8(rValueX)

		gValueX, err := strconv.Atoi(numbers[gIndex])
		if err != nil {
			return -1, nil, err
		}
		gValue := uint8(gValueX)

		bValueX, err := strconv.Atoi(numbers[bIndex])
		if err != nil {
			return -1, nil, err
		}
		bValue := uint8(bValueX)

		colorValue := twin.NewColor24Bit(rValue, gValue, bValue)
		return bIndex + 1, &colorValue, nil
	}

	err := fmt.Errorf(
		"unknown color type <%s>, expected 5 (8 bit color) or 2 (24 bit color): <CSI %sm>",
		numbers[index],
		strings.Join(numbers[baseIndex:], ";"))
	return -1, nil, err
}
