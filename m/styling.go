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

// From LESS_TERMCAP_so, overrides statusbarStyle from the Chroma style if set.
// Used for overriding statusbarStyle.
var standoutStyle *twin.Style

var lineNumbersStyle = twin.StyleDefault.WithAttr(twin.AttrDim)
var statusbarStyle = twin.StyleDefault.WithAttr(twin.AttrReverse)

func setStyle(updateMe *twin.Style, envVarName string, fallback *twin.Style) {
	envValue := os.Getenv(envVarName)
	if envValue == "" {
		if fallback != nil {
			*updateMe = *fallback
		}
		return
	}

	*updateMe = termcapToStyle(envValue)
}

func twinStyleFromChroma(chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter, chromaToken chroma.TokenType) *twin.Style {
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
	cells := textstyles.CellsFromString(formatted, nil).Cells
	if len(cells) != 1 {
		log.Warnf("Chroma formatter didn't return exactly one cell: %#v", cells)
		return nil
	}

	return &cells[0].Style
}

// consumeLessTermcapEnvs parses LESS_TERMCAP_xx environment variables and
// adapts the moar output accordingly.
func consumeLessTermcapEnvs(chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter) {
	// Requested here: https://github.com/walles/moar/issues/14

	setStyle(&textstyles.ManPageBold, "LESS_TERMCAP_md", twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.GenericStrong))
	setStyle(&textstyles.ManPageUnderline, "LESS_TERMCAP_us", twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.GenericUnderline))

	// Since standoutStyle defaults to nil we can't just pass it to setStyle().
	// Instead we give it special treatment here and set it only if its
	// environment variable is set.
	//
	// Ref: https://github.com/walles/moar/issues/171
	envValue := os.Getenv("LESS_TERMCAP_so")
	if envValue != "" {
		style := termcapToStyle(envValue)
		standoutStyle = &style
	}
}

func styleUI(chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter, statusbarOption StatusBarOption) {
	if chromaStyle == nil || chromaFormatter == nil {
		return
	}

	chromaLineNumbers := twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.LineNumbers)
	if chromaLineNumbers != nil {
		// If somebody can provide an example where not-dimmed line numbers
		// looks good I'll change this, but until then they will be dimmed no
		// matter what the theme authors think.
		lineNumbersStyle = chromaLineNumbers.WithAttr(twin.AttrDim)
	}

	if standoutStyle != nil {
		statusbarStyle = *standoutStyle
	} else if statusbarOption == STATUSBAR_STYLE_INVERSE {
		// FIXME: Get this from the Chroma style
		statusbarStyle = twin.StyleDefault.WithAttr(twin.AttrReverse)
	} else if statusbarOption == STATUSBAR_STYLE_PLAIN {
		plain := twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.None)
		if plain != nil {
			statusbarStyle = *plain
		} else {
			statusbarStyle = twin.StyleDefault
		}
	} else if statusbarOption == STATUSBAR_STYLE_BOLD {
		bold := twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.GenericStrong)
		if bold != nil {
			statusbarStyle = *bold
		} else {
			statusbarStyle = twin.StyleDefault.WithAttr(twin.AttrBold)
		}
	} else {
		panic(fmt.Sprint("Unrecognized status bar style: ", statusbarOption))
	}
}

func termcapToStyle(termcap string) twin.Style {
	// Add a character to be sure we have one to take the format from
	cells := textstyles.CellsFromString(termcap+"x", nil).Cells
	return cells[len(cells)-1].Style
}
