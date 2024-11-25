package m

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
)

// From LESS_TERMCAP_so, overrides statusbarStyle from the Chroma style if set
var standoutStyle *twin.Style

var lineNumbersStyle = twin.StyleDefault.WithAttr(twin.AttrDim)
var statusbarStyle = twin.StyleDefault.WithAttr(twin.AttrReverse)

var plainTextStyle = twin.StyleDefault

func setStyle(updateMe *twin.Style, envVarName string, fallback *twin.Style) {
	envValue := os.Getenv(envVarName)
	if envValue == "" {
		if fallback != nil {
			*updateMe = *fallback
		}
		return
	}

	style, err := TermcapToStyle(envValue)
	if err != nil {
		log.Debug("Ignoring invalid ", envVarName, ": ", strings.ReplaceAll(envValue, "\x1b", "ESC"), ": ", err)
		return
	}

	*updateMe = style
}

// With exact set, only return a style if the Chroma formatter has an explicit
// configuration for that style. Otherwise, we might return fallback styles, not
// exactly matching what you requested.
func twinStyleFromChroma(chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter, chromaToken chroma.TokenType, exact bool) *twin.Style {
	if chromaStyle == nil || chromaFormatter == nil {
		return nil
	}

	stringBuilder := strings.Builder{}
	err := (*chromaFormatter).Format(&stringBuilder, chromaStyle, chroma.Literator(chroma.Token{
		Type:  chromaToken,
		Value: "X",
	}))
	if err != nil {
		panic(err)
	}

	formatted := stringBuilder.String()
	cells := textstyles.StyledRunesFromString(twin.StyleDefault, formatted, nil).StyledRunes
	if len(cells) != 1 {
		log.Warnf("Chroma formatter didn't return exactly one cell: %#v", cells)
		return nil
	}

	inexactStyle := cells[0].Style
	if !exact {
		return &inexactStyle
	}

	unstyled := twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.None, false)
	if unstyled == nil {
		panic("Chroma formatter didn't return a style for chroma.None")
	}
	if inexactStyle != *unstyled {
		// We got something other than the style of None, return it!
		return &inexactStyle
	}

	return nil
}

// consumeLessTermcapEnvs parses LESS_TERMCAP_xx environment variables and
// adapts the moar output accordingly.
func consumeLessTermcapEnvs(chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter) {
	// Requested here: https://github.com/walles/moar/issues/14

	setStyle(
		&textstyles.ManPageBold,
		"LESS_TERMCAP_md",
		twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.GenericStrong, false),
	)
	setStyle(&textstyles.ManPageUnderline,
		"LESS_TERMCAP_us",
		twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.GenericUnderline, false),
	)

	// Since standoutStyle defaults to nil we can't just pass it to setStyle().
	// Instead we give it special treatment here and set it only if its
	// environment variable is set.
	//
	// Ref: https://github.com/walles/moar/issues/171
	envValue := os.Getenv("LESS_TERMCAP_so")
	if envValue != "" {
		style, err := TermcapToStyle(envValue)
		if err == nil {
			standoutStyle = &style
		} else {
			log.Debug("Ignoring invalid LESS_TERMCAP_so: ", strings.ReplaceAll(envValue, "\x1b", "ESC"), ": ", err)
		}
	}
}

func styleUI(chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter, statusbarOption StatusBarOption, withTerminalFg bool) {
	if chromaStyle == nil || chromaFormatter == nil {
		return
	}

	headingStyle := twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.GenericHeading, true)
	if headingStyle != nil {
		textstyles.ManPageHeading = *headingStyle
	}

	chromaLineNumbers := twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.LineNumbers, true)
	if chromaLineNumbers != nil {
		// NOTE: We used to dim line numbers here, but Johan found them too hard
		// to read. If line numbers should look some other way for some Chroma
		// style, go fix that in Chroma!
		lineNumbersStyle = *chromaLineNumbers
	}

	if withTerminalFg {
		plainTextStyle = twin.StyleDefault
	} else {
		plainText := twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.None, false)
		if plainText != nil {
			plainTextStyle = *plainText
		}
	}

	if standoutStyle != nil {
		statusbarStyle = *standoutStyle
	} else if statusbarOption == STATUSBAR_STYLE_INVERSE {
		// FIXME: Get this from the Chroma style
		statusbarStyle = twin.StyleDefault.WithAttr(twin.AttrReverse)
	} else if statusbarOption == STATUSBAR_STYLE_PLAIN {
		plain := twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.None, false)
		if plain != nil {
			statusbarStyle = *plain
		} else {
			statusbarStyle = twin.StyleDefault
		}
	} else if statusbarOption == STATUSBAR_STYLE_BOLD {
		bold := twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.GenericStrong, true)
		if bold != nil {
			statusbarStyle = *bold
		} else {
			statusbarStyle = twin.StyleDefault.WithAttr(twin.AttrBold)
		}
	} else {
		panic(fmt.Sprint("Unrecognized status bar style: ", statusbarOption))
	}
}

func TermcapToStyle(termcap string) (twin.Style, error) {
	// Add a character to be sure we have one to take the format from
	cells := textstyles.StyledRunesFromString(twin.StyleDefault, termcap+"x", nil).StyledRunes
	if len(cells) != 1 {
		return twin.StyleDefault, fmt.Errorf("Expected styling only and no text")
	}
	return cells[len(cells)-1].Style, nil
}
