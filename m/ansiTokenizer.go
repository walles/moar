package m

import (
	"regexp"
	"strings"

	"github.com/gdamore/tcell"
)

type Token struct {
	Rune  rune
	Style tcell.Style
}

// TokensFromString turns a string into a series of tokens
func TokensFromString(s string) []Token {
	var tokens []Token

	for _, styledString := range _StyledStringsFromString(s) {
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

func _StyledStringsFromString(s string) []_StyledString {
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
		style = _UpdateStyle(style, matchedPart)

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
func _UpdateStyle(style tcell.Style, escapeSequence string) tcell.Style {
	for _, number := range strings.Split(escapeSequence[2:len(escapeSequence)-1], ";") {
		switch number {
		case "0":
			style = tcell.StyleDefault
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
		}
	}

	return style
}
