package m

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/gdamore/tcell"
)

const _TabSize = 4

var manPageBold = tcell.StyleDefault.Bold(true)
var manPageUnderline = tcell.StyleDefault.Underline(true)

// Token is a rune with a style to be written to a cell on screen
type Token struct {
	Rune  rune
	Style tcell.Style
}

// SetManPageFormatFromEnv parses LESS_TERMCAP_xx environment variables and
// adapts the moar output accordingly.
func SetManPageFormatFromEnv(logger *log.Logger) {
	// Requested here: https://github.com/walles/moar/issues/14

	lessTermcapMd := os.Getenv("LESS_TERMCAP_md")
	if lessTermcapMd != "" {
		manPageBold = _TermcapToStyle(logger, lessTermcapMd)
	}

	lessTermcapUs := os.Getenv("LESS_TERMCAP_us")
	if lessTermcapUs != "" {
		manPageUnderline = _TermcapToStyle(logger, lessTermcapUs)
	}
}

// Used from tests
func _ResetManPageFormatForTesting() {
	manPageBold = tcell.StyleDefault.Bold(true)
	manPageUnderline = tcell.StyleDefault.Underline(true)
}

func _TermcapToStyle(logger *log.Logger, termcap string) tcell.Style {
	// Add a character to be sure we have one to take the format from
	tokens, _ := TokensFromString(logger, termcap+"x")
	return tokens[len(tokens)-1].Style
}

// TokensFromString turns a (formatted) string into a series of tokens,
// and an unformatted string
func TokensFromString(logger *log.Logger, s string) ([]Token, *string) {
	var tokens []Token

	styleBrokenUtf8 := tcell.StyleDefault.Background(7).Foreground(1)

	for _, styledString := range _StyledStringsFromString(logger, s) {
		for _, token := range _TokensFromStyledString(styledString) {
			switch token.Rune {

			case '\x09': // TAB
				for {
					tokens = append(tokens, Token{
						Rune:  ' ',
						Style: styledString.Style,
					})

					if (len(tokens))%_TabSize == 0 {
						// We arrived at the next tab stop
						break
					}
				}

			case '�': // Go's broken-UTF8 marker
				tokens = append(tokens, Token{
					Rune:  '?',
					Style: styleBrokenUtf8,
				})

			case '\x08': // Backspace
				tokens = append(tokens, Token{
					Rune:  '<',
					Style: styleBrokenUtf8,
				})

			default:
				tokens = append(tokens, token)
			}
		}
	}

	var stringBuilder strings.Builder
	stringBuilder.Grow(len(tokens))
	for _, token := range tokens {
		stringBuilder.WriteRune(token.Rune)
	}
	plainString := stringBuilder.String()
	return tokens, &plainString
}

func _TokensFromStyledString(styledString _StyledString) []Token {
	tokens := make([]Token, 0, len(styledString.String)+1)
	oneBack := '\x00'
	twoBack := '\x00'
	for _, char := range []rune(styledString.String) {
		if oneBack == '\x08' && twoBack != '\x00' {
			// Something-Backspace-Something

			replacement := (*Token)(nil)

			if char == twoBack {
				replacement = &Token{
					Rune:  twoBack,
					Style: manPageBold,
				}
			}

			if twoBack == '_' {
				replacement = &Token{
					Rune:  char,
					Style: manPageUnderline,
				}
			}

			// FIXME: Man page formatting fails, if I do (in bash)...
			//   "man printf|hexdump -C|grep -10 leading| grep --color 08"
			// ... I get...
			//   "000003e0  20 20 20 20 20 20 20 20  2b 08 2b 08 6f 08 6f 20  |        +.+.o.o |"
			// ... wich "less" renders as a bold "o". We should as well.
			//
			// I don't get the logic though, the sequence is:
			//   plus-backspace-plus-backspace-o-backspace-o
			//
			// Maybe the interpretation should be:
			//   "Make a bold +, then erase that and replace it with a bold o"?
			//
			// Used for bullet points, maybe we should just replace the whole thing with
			// a unicode bullet point in bold?

			if replacement != nil {
				tokens = append(tokens[0:len(tokens)-2], *replacement)

				twoBack = oneBack
				oneBack = char
				continue
			}

			// No match, just keep going
		}

		tokens = append(tokens, Token{
			Rune:  char,
			Style: styledString.Style,
		})

		twoBack = oneBack
		oneBack = char
	}

	return tokens
}

type _StyledString struct {
	String string
	Style  tcell.Style
}

func _StyledStringsFromString(logger *log.Logger, s string) []_StyledString {
	// This function was inspired by the
	// https://golang.org/pkg/regexp/#Regexp.Split source code

	pattern := regexp.MustCompile("\x1b\\[([0-9;]*m)")

	matches := pattern.FindAllStringIndex(s, -1)
	styledStrings := make([]_StyledString, 0, len(matches)+1)

	style := tcell.StyleDefault

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
		style = _UpdateStyle(logger, style, matchedPart)

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

// _UpdateStyle parses a string of the form "ESC[33m" into changes to style
func _UpdateStyle(logger *log.Logger, style tcell.Style, escapeSequence string) tcell.Style {
	numbers := strings.Split(escapeSequence[2:len(escapeSequence)-1], ";")
	index := 0
	for index < len(numbers) {
		number := numbers[index]
		index++
		switch strings.TrimLeft(number, "0") {
		case "":
			style = tcell.StyleDefault

		case "1":
			style = style.Bold(true)

		case "4":
			style = style.Underline(true)

		case "7":
			style = style.Reverse(true)

		case "27":
			style = style.Reverse(false)

		// Foreground colors
		case "30":
			style = style.Foreground(0)
		case "31":
			style = style.Foreground(1)
		case "32":
			style = style.Foreground(2)
		case "33":
			style = style.Foreground(3)
		case "34":
			style = style.Foreground(4)
		case "35":
			style = style.Foreground(5)
		case "36":
			style = style.Foreground(6)
		case "37":
			style = style.Foreground(7)
		case "38":
			var err error = nil
			var color *tcell.Color
			index, color, err = consumeCompositeColor(numbers, index-1)
			if err != nil {
				logger.Printf("Foreground: %s", err.Error())
				return style
			}
			style = style.Foreground(*color)
		case "39":
			style = style.Foreground(tcell.ColorDefault)

		// Background colors
		case "40":
			style = style.Background(0)
		case "41":
			style = style.Background(1)
		case "42":
			style = style.Background(2)
		case "43":
			style = style.Background(3)
		case "44":
			style = style.Background(4)
		case "45":
			style = style.Background(5)
		case "46":
			style = style.Background(6)
		case "47":
			style = style.Background(7)
		case "48":
			var err error = nil
			var color *tcell.Color
			index, color, err = consumeCompositeColor(numbers, index-1)
			if err != nil {
				logger.Printf("Background: %s", err.Error())
				return style
			}
			style = style.Background(*color)
		case "49":
			style = style.Background(tcell.ColorDefault)

		default:
			logger.Printf("Unrecognized ANSI SGR code <%s>", number)
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
func consumeCompositeColor(numbers []string, index int) (int, *tcell.Color, error) {
	baseIndex := index
	if numbers[index] != "38" && numbers[index] != "48" {
		err := fmt.Errorf(
			"Unknown start of color sequence <%s>, expected 38 (foreground) or 48 (background): <CSI %sm>",
			numbers[index],
			strings.Join(numbers[baseIndex:], ";"))
		return -1, nil, err
	}

	index++
	if index >= len(numbers) {
		err := fmt.Errorf(
			"Incomplete color sequence: <CSI %sm>",
			strings.Join(numbers[baseIndex:], ";"))
		return -1, nil, err
	}

	if numbers[index] == "5" {
		// Handle 8 bit color
		index++
		if index >= len(numbers) {
			err := fmt.Errorf(
				"Incomplete 8 bit color sequence: <CSI %sm>",
				strings.Join(numbers[baseIndex:], ";"))
			return -1, nil, err
		}

		colorNumber, err := strconv.Atoi(numbers[index])
		if err != nil {
			return -1, nil, err
		}

		colorValue := tcell.Color(colorNumber)
		return index + 1, &colorValue, nil
	}

	if numbers[index] == "2" {
		// Handle 24 bit color
		rIndex := index + 1
		gIndex := index + 2
		bIndex := index + 3
		if bIndex >= len(numbers) {
			err := fmt.Errorf(
				"Incomplete 24 bit color sequence, expected N8;2;R;G;Bm: <CSI %sm>",
				strings.Join(numbers[baseIndex:], ";"))
			return -1, nil, err
		}

		rValueX, err := strconv.ParseInt(numbers[rIndex], 10, 32)
		if err != nil {
			return -1, nil, err
		}
		rValue := int32(rValueX)

		gValueX, err := strconv.Atoi(numbers[gIndex])
		if err != nil {
			return -1, nil, err
		}
		gValue := int32(gValueX)

		bValueX, err := strconv.Atoi(numbers[bIndex])
		if err != nil {
			return -1, nil, err
		}
		bValue := int32(bValueX)

		colorValue := tcell.NewRGBColor(rValue, gValue, bValue)
		return bIndex + 1, &colorValue, nil
	}

	err := fmt.Errorf(
		"Unknown color type <%s>, expected 5 (8 bit color) or 2 (24 bit color): <CSI %sm>",
		numbers[index],
		strings.Join(numbers[baseIndex:], ";"))
	return -1, nil, err
}
