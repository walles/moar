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
	fg    Color
	bg    Color
	attrs AttrMask

	// This hyperlinkUrl is a URL for in-terminal hyperlinks.
	//
	// Since we don't want to do error handling of broken URLs, we just store
	// these URLs as strings.
	//
	// Ref:
	// * https://gist.github.com/egmontkob/eb114294efbcd5adb1944c9f3cb5feda
	// * https://github.com/walles/moar/issues/131
	hyperlinkUrl *string
}

var StyleDefault Style

func (style Style) String() string {
	if style.attrs == AttrNone {
		return fmt.Sprint(style.fg, " on ", style.bg)
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

	return fmt.Sprint(strings.Join(attrNames, " "), " ", style.fg, " on ", style.bg)
}

func (style Style) WithAttr(attr AttrMask) Style {
	result := Style{
		fg:           style.fg,
		bg:           style.bg,
		attrs:        style.attrs | attr,
		hyperlinkUrl: style.hyperlinkUrl,
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
func (style Style) WithHyperlink(hyperlinkUrl *string) Style {
	return Style{
		fg:           style.fg,
		bg:           style.bg,
		attrs:        style.attrs,
		hyperlinkUrl: hyperlinkUrl,
	}
}

func (style Style) WithoutAttr(attr AttrMask) Style {
	return Style{
		fg:           style.fg,
		bg:           style.bg,
		attrs:        style.attrs & ^attr,
		hyperlinkUrl: style.hyperlinkUrl,
	}
}

func (attr AttrMask) has(attrs AttrMask) bool {
	return attr&attrs != 0
}

func (style Style) Background(color Color) Style {
	return Style{
		fg:           style.fg,
		bg:           color,
		attrs:        style.attrs,
		hyperlinkUrl: style.hyperlinkUrl,
	}
}

func (style Style) Foreground(color Color) Style {
	return Style{
		fg:           color,
		bg:           style.bg,
		attrs:        style.attrs,
		hyperlinkUrl: style.hyperlinkUrl,
	}
}

// Emit an ANSI escape sequence switching from a previous style to the current
// one.
func (current Style) RenderUpdateFrom(previous Style) string {
	var builder strings.Builder
	if current.fg != previous.fg {
		builder.WriteString(current.fg.ForegroundAnsiString())
	}

	if current.bg != previous.bg {
		builder.WriteString(current.bg.BackgroundAnsiString())
	}

	// Handle AttrDim / AttrBold changes
	previousBoldDim := previous.attrs & (AttrBold | AttrDim)
	currentBoldDim := current.attrs & (AttrBold | AttrDim)
	if currentBoldDim != previousBoldDim {
		if previousBoldDim != 0 {
			builder.WriteString("\x1b[22m") // Reset to neither bold nor dim
		}
		if current.attrs.has(AttrBold) {
			builder.WriteString("\x1b[1m")
		}
		if current.attrs.has(AttrDim) {
			builder.WriteString("\x1b[2m")
		}
	}

	// Handle AttrBlink changes
	if current.attrs.has(AttrBlink) != previous.attrs.has(AttrBlink) {
		if current.attrs.has(AttrBlink) {
			builder.WriteString("\x1b[5m")
		} else {
			builder.WriteString("\x1b[25m")
		}
	}

	// Handle AttrReverse changes
	if current.attrs.has(AttrReverse) != previous.attrs.has(AttrReverse) {
		if current.attrs.has(AttrReverse) {
			builder.WriteString("\x1b[7m")
		} else {
			builder.WriteString("\x1b[27m")
		}
	}

	// Handle AttrUnderline changes
	if current.attrs.has(AttrUnderline) != previous.attrs.has(AttrUnderline) {
		if current.attrs.has(AttrUnderline) {
			builder.WriteString("\x1b[4m")
		} else {
			builder.WriteString("\x1b[24m")
		}
	}

	// Handle AttrItalic changes
	if current.attrs.has(AttrItalic) != previous.attrs.has(AttrItalic) {
		if current.attrs.has(AttrItalic) {
			builder.WriteString("\x1b[3m")
		} else {
			builder.WriteString("\x1b[23m")
		}
	}

	// Handle AttrStrikeThrough changes
	if current.attrs.has(AttrStrikeThrough) != previous.attrs.has(AttrStrikeThrough) {
		if current.attrs.has(AttrStrikeThrough) {
			builder.WriteString("\x1b[9m")
		} else {
			builder.WriteString("\x1b[29m")
		}
	}

	if current.hyperlinkUrl != previous.hyperlinkUrl {
		newUrl := ""
		if current.hyperlinkUrl != nil {
			newUrl = *current.hyperlinkUrl
		}
		builder.WriteString("\x1b]8;;")
		builder.WriteString(newUrl)
		builder.WriteString("\x1b\\")
	}

	return builder.String()
}
