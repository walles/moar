package m

import (
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/twin"
)

// From LESS_TERMCAP_so, overrides statusbarStyle from the Chroma style if set
var standoutStyle *twin.Style = nil

var manPageBold = twin.StyleDefault.WithAttr(twin.AttrBold)
var manPageUnderline = twin.StyleDefault.WithAttr(twin.AttrUnderline)

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
	cells := cellsFromString(formatted, nil).Cells
	if len(cells) != 1 {
		log.Warnf("Chroma formatter didn't return exactly one cell: %#v", cells)
		return nil
	}

	return &cells[0].Style
}

func backgroundStyleFromChroma(chromaStyle *chroma.Style) *twin.Style {
	if chromaStyle == nil {
		return nil
	}

	backgroundEntry := chromaStyle.Get(chroma.Background)

	if !backgroundEntry.Background.IsSet() {
		panic(fmt.Sprint("Background color not set in style: ", chromaStyle))
	}
	backgroundColor := twin.NewColor24Bit(
		backgroundEntry.Background.Red(),
		backgroundEntry.Background.Green(),
		backgroundEntry.Background.Blue())

	foregroundColor := twin.ColorDefault
	if backgroundEntry.Colour.IsSet() {
		foregroundColor = twin.NewColor24Bit(
			backgroundEntry.Colour.Red(),
			backgroundEntry.Colour.Green(),
			backgroundEntry.Colour.Blue())
	}

	returnMe := twin.StyleDefault.
		Background(backgroundColor).
		Foreground(foregroundColor)

	if backgroundEntry.Bold == chroma.Yes {
		returnMe = returnMe.WithAttr(twin.AttrBold)
	}
	if backgroundEntry.Italic == chroma.Yes {
		returnMe = returnMe.WithAttr(twin.AttrItalic)
	}
	if backgroundEntry.Underline == chroma.Yes {
		returnMe = returnMe.WithAttr(twin.AttrUnderline)
	}

	return &returnMe
}

// consumeLessTermcapEnvs parses LESS_TERMCAP_xx environment variables and
// adapts the moar output accordingly.
func consumeLessTermcapEnvs(chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter) {
	// Requested here: https://github.com/walles/moar/issues/14

	setStyle(&manPageBold, "LESS_TERMCAP_md", twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.GenericStrong))
	setStyle(&manPageUnderline, "LESS_TERMCAP_us", twinStyleFromChroma(chromaStyle, chromaFormatter, chroma.GenericUnderline))

	// Special treat this because standoutStyle defaults to nil, and should be
	// set only if there is a style defined through the environment.
	envValue := os.Getenv("LESS_TERMCAP_so")
	if envValue != "" {
		style := termcapToStyle(envValue)
		standoutStyle = &style
	}
}

func styleUi(chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter, statusbarOption StatusBarOption) {
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
		styleBackground := backgroundStyleFromChroma(chromaStyle)
		if styleBackground != nil {
			statusbarStyle = styleBackground.WithAttr(twin.AttrReverse)
		} else {
			statusbarStyle = twin.StyleDefault.WithAttr(twin.AttrReverse)
		}
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
	cells := cellsFromString(termcap+"x", nil).Cells
	return cells[len(cells)-1].Style
}
