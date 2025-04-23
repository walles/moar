package twin

import (
	"fmt"
	"strings"
)

type AttrMask uint

const (
	AttrBold AttrMask = 1 << iota
	AttrBlink
	AttrReverse
	AttrUnderline
	AttrDim
	AttrItalic
	AttrStrikeThrough
	AttrNone AttrMask = 0 // Normal text
)

type Style struct {
	fg             Color
	bg             Color
	underlineColor Color
	attrs          AttrMask

	// This hyperlinkURL is a URL for in-terminal hyperlinks.
	//
	// Since we don't want to do error handling of broken URLs, we just store
	// these URLs as strings.
	//
	// Ref:
	// * https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda
	// * https://github.com/walles/moar/issues/131
	hyperlinkURL *string
}

var StyleDefault Style

func (style Style) String() string {
	undelineSuffix := ""
	if style.underlineColor != ColorDefault {
		undelineSuffix = fmt.Sprintf(" underlined with %v", style.underlineColor)
	}

	attrNames := make([]string, 0)
	if style.attrs.has(AttrBold) {
		attrNames = append(attrNames, "bold")
	}
	if style.attrs.has(AttrBlink) {
		attrNames = append(attrNames, "blinking")
	}
	if style.attrs.has(AttrReverse) {
		attrNames = append(attrNames, "reverse")
	}
	if style.attrs.has(AttrUnderline) {
		attrNames = append(attrNames, "underlined")
	}
	if style.attrs.has(AttrDim) {
		attrNames = append(attrNames, "dim")
	}
	if style.attrs.has(AttrItalic) {
		attrNames = append(attrNames, "italic")
	}
	if style.attrs.has(AttrStrikeThrough) {
		attrNames = append(attrNames, "strikethrough")
	}
	if style.hyperlinkURL != nil {
		attrNames = append(attrNames, "\""+*style.hyperlinkURL+"\"")
	}

	if len(attrNames) == 0 {
		return fmt.Sprint(style.fg, " on ", style.bg, undelineSuffix)
	}

	return fmt.Sprint(strings.Join(attrNames, " "), " ", style.fg, " on ", style.bg, undelineSuffix)
}

func (style Style) WithAttr(attr AttrMask) Style {
	result := Style{
		fg:             style.fg,
		bg:             style.bg,
		underlineColor: style.underlineColor,
		attrs:          style.attrs | attr,
		hyperlinkURL:   style.hyperlinkURL,
	}

	// Bold and dim are mutually exclusive
	if attr.has(AttrBold) {
		return result.WithoutAttr(AttrDim)
	}
	if attr.has(AttrDim) {
		return result.WithoutAttr(AttrBold)
	}

	return result
}

// Call with nil to remove the link
func (style Style) WithHyperlink(hyperlinkURL *string) Style {
	if hyperlinkURL != nil && *hyperlinkURL == "" {
		// Use nil instead of empty string
		hyperlinkURL = nil
	}

	return Style{
		fg:             style.fg,
		bg:             style.bg,
		underlineColor: style.underlineColor,
		attrs:          style.attrs,
		hyperlinkURL:   hyperlinkURL,
	}
}

func (style Style) WithoutAttr(attr AttrMask) Style {
	return Style{
		fg:             style.fg,
		bg:             style.bg,
		underlineColor: style.underlineColor,
		attrs:          style.attrs & ^attr,
		hyperlinkURL:   style.hyperlinkURL,
	}
}

func (attr AttrMask) has(attrs AttrMask) bool {
	return attr&attrs != 0
}

func (style Style) WithBackground(color Color) Style {
	return Style{
		fg:             style.fg,
		bg:             color,
		underlineColor: style.underlineColor,
		attrs:          style.attrs,
		hyperlinkURL:   style.hyperlinkURL,
	}
}

func (style Style) WithForeground(color Color) Style {
	return Style{
		fg:             color,
		bg:             style.bg,
		underlineColor: style.underlineColor,
		attrs:          style.attrs,
		hyperlinkURL:   style.hyperlinkURL,
	}
}

func (style Style) WithUnderlineColor(color Color) Style {
	return Style{
		fg:             style.fg,
		bg:             style.bg,
		underlineColor: color,
		attrs:          style.attrs,
		hyperlinkURL:   style.hyperlinkURL,
	}
}

// Emit an ANSI escape sequence switching from a previous style to the current
// one.
//
//revive:disable-next-line:receiver-naming
func (style Style) RenderUpdateFrom(previous Style, terminalColorCount ColorCount) string {
	if style == previous {
		// Shortcut for the common case
		return ""
	}

	hadHyperlink := previous.hyperlinkURL != nil && *previous.hyperlinkURL != ""
	if style == StyleDefault && !hadHyperlink {
		return "\x1b[m"
	}

	var builder strings.Builder
	if style.fg != previous.fg {
		builder.WriteString(style.fg.ansiString(colorTypeForeground, terminalColorCount))
	}

	if style.bg != previous.bg {
		builder.WriteString(style.bg.ansiString(colorTypeBackground, terminalColorCount))
	}

	if style.underlineColor != previous.underlineColor {
		builder.WriteString(style.underlineColor.ansiString(colorTypeUnderline, terminalColorCount))
	}

	// Handle AttrDim / AttrBold changes
	previousBoldDim := previous.attrs & (AttrBold | AttrDim)
	currentBoldDim := style.attrs & (AttrBold | AttrDim)
	if currentBoldDim != previousBoldDim {
		if previousBoldDim != 0 {
			builder.WriteString("\x1b[22m") // Reset to neither bold nor dim
		}
		if style.attrs.has(AttrBold) {
			builder.WriteString("\x1b[1m")
		}
		if style.attrs.has(AttrDim) {
			builder.WriteString("\x1b[2m")
		}
	}

	// Handle AttrBlink changes
	if style.attrs.has(AttrBlink) != previous.attrs.has(AttrBlink) {
		if style.attrs.has(AttrBlink) {
			builder.WriteString("\x1b[5m")
		} else {
			builder.WriteString("\x1b[25m")
		}
	}

	// Handle AttrReverse changes
	if style.attrs.has(AttrReverse) != previous.attrs.has(AttrReverse) {
		if style.attrs.has(AttrReverse) {
			builder.WriteString("\x1b[7m")
		} else {
			builder.WriteString("\x1b[27m")
		}
	}

	// Handle AttrUnderline changes
	if style.attrs.has(AttrUnderline) != previous.attrs.has(AttrUnderline) {
		if style.attrs.has(AttrUnderline) {
			builder.WriteString("\x1b[4m")
		} else {
			builder.WriteString("\x1b[24m")
		}
	}

	// Handle AttrItalic changes
	if style.attrs.has(AttrItalic) != previous.attrs.has(AttrItalic) {
		if style.attrs.has(AttrItalic) {
			builder.WriteString("\x1b[3m")
		} else {
			builder.WriteString("\x1b[23m")
		}
	}

	// Handle AttrStrikeThrough changes
	if style.attrs.has(AttrStrikeThrough) != previous.attrs.has(AttrStrikeThrough) {
		if style.attrs.has(AttrStrikeThrough) {
			builder.WriteString("\x1b[9m")
		} else {
			builder.WriteString("\x1b[29m")
		}
	}

	if style.hyperlinkURL != previous.hyperlinkURL {
		newURL := ""
		if style.hyperlinkURL != nil {
			newURL = *style.hyperlinkURL
		}

		previousURL := ""
		if previous.hyperlinkURL != nil {
			previousURL = *previous.hyperlinkURL
		}

		if newURL != previousURL {
			builder.WriteString("\x1b]8;;")
			builder.WriteString(newURL)
			builder.WriteString("\x1b\\")
		}
	}

	return builder.String()
}
