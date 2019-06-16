package m

import (
	"log"
	"regexp"
	"strings"

	"github.com/gdamore/tcell"
)

// Token is a rune with a style to be written to a cell on screen
type Token struct {
	Rune  rune
	Style tcell.Style
}

// TokensFromString turns a string into a series of tokens
func TokensFromString(logger *log.Logger, s string) []Token {
	var tokens []Token

	for _, styledString := range _StyledStringsFromString(logger, s) {
		for _, char := range styledString.String {
			tokens = append(tokens, Token{
				Rune:  char,
				Style: styledString.Style,
			})
		}
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
			// Otherwise the string is empty, no point for us in that
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
	for _, number := range strings.Split(escapeSequence[2:len(escapeSequence)-1], ";") {
		switch number {
		case "", "0":
			style = tcell.StyleDefault

		case "1":
			style = style.Bold(true)

		// Foreground colors
		case "30":
			style = style.Foreground(tcell.ColorBlack)
		case "31":
			style = style.Foreground(tcell.ColorRed)
		case "32":
			style = style.Foreground(tcell.ColorGreen)
		case "33":
			style = style.Foreground(tcell.ColorYellow)
		case "34":
			style = style.Foreground(tcell.ColorBlue)
		case "35":
			style = style.Foreground(tcell.ColorPurple)
		case "36":
			style = style.Foreground(tcell.ColorTeal)
		case "37":
			style = style.Foreground(tcell.ColorWhite)

		// Background colors
		case "40":
			style = style.Background(tcell.ColorBlack)
		case "41":
			style = style.Background(tcell.ColorRed)
		case "42":
			style = style.Background(tcell.ColorGreen)
		case "43":
			style = style.Background(tcell.ColorYellow)
		case "44":
			style = style.Background(tcell.ColorBlue)
		case "45":
			style = style.Background(tcell.ColorPurple)
		case "46":
			style = style.Background(tcell.ColorTeal)
		case "47":
			style = style.Background(tcell.ColorWhite)

		default:
			logger.Printf("Unrecognized ANSI SGI code <%s>", number)
		}
	}

	return style
}
