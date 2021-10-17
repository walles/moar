package twin

import "fmt"

// Create using NewColor16(), NewColor256 or NewColor24Bit(), or use
// ColorDefault.
type Color uint32
type ColorType uint8

const (
	// Default foreground / background color
	colorTypeDefault ColorType = iota

	// https://en.wikipedia.org/wiki/ANSI_escape_code#3-bit_and_4-bit
	colorType16

	// https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit
	colorType256

	// RGB: https://en.wikipedia.org/wiki/ANSI_escape_code#24-bit
	colorType24bit
)

// Reset to default foreground / background color
var ColorDefault = newColor(colorTypeDefault, 0)

// From: https://en.wikipedia.org/wiki/ANSI_escape_code#3-bit_and_4-bit
var colorNames16 = map[int]string{
	0:  "0 black",
	1:  "1 red",
	2:  "2 green",
	3:  "3 yellow (orange)",
	4:  "4 blue",
	5:  "5 magenta",
	6:  "6 cyan",
	7:  "7 white (light gray)",
	8:  "8 bright black (dark gray)",
	9:  "9 bright red",
	10: "10 bright green",
	11: "11 bright yellow",
	12: "12 bright blue",
	13: "13 bright magenta",
	14: "14 bright cyan",
	15: "15 bright white",
}

func newColor(colorType ColorType, value uint32) Color {
	return Color(value | (uint32(colorType) << 24))
}

// Four bit colors as defined here:
// https://en.wikipedia.org/wiki/ANSI_escape_code#3-bit_and_4-bit
func NewColor16(colorNumber0to15 int) Color {
	return newColor(colorType16, uint32(colorNumber0to15))
}

func NewColor256(colorNumber uint8) Color {
	return newColor(colorType256, uint32(colorNumber))
}

func NewColor24Bit(red uint8, green uint8, blue uint8) Color {
	return newColor(colorType24bit, (uint32(red)<<16)+(uint32(green)<<8)+(uint32(blue)<<0))
}

func NewColorHex(rgb uint32) Color {
	return newColor(colorType24bit, rgb)
}

func (color Color) colorType() ColorType {
	return ColorType(color >> 24)
}

func (color Color) colorValue() uint32 {
	return uint32(color & 0xff_ff_ff)
}

// Render color into an ANSI string.
//
// Ref: https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters
func (color Color) ansiString(foreground bool) string {
	value := color.colorValue()

	fgBgMarker := "3"
	if !foreground {
		fgBgMarker = "4"
	}

	if color.colorType() == colorType16 {
		if value < 8 {
			return fmt.Sprint("\x1b[", fgBgMarker, value, "m")
		} else if value <= 15 {
			fgBgMarker := "9"
			if !foreground {
				fgBgMarker = "10"
			}
			return fmt.Sprint("\x1b[", fgBgMarker, value-8, "m")
		}
	}

	if color.colorType() == colorType256 {
		if value <= 255 {
			return fmt.Sprint("\x1b[", fgBgMarker, "8;5;", value, "m")
		}
	}

	if color.colorType() == colorType24bit {
		red := (value & 0xff0000) >> 16
		green := (value & 0xff00) >> 8
		blue := value & 0xff

		return fmt.Sprint("\x1b[", fgBgMarker, "8;2;", red, ";", green, ";", blue, "m")
	}

	if color.colorType() == colorTypeDefault {
		return fmt.Sprint("\x1b[", fgBgMarker, "9m")
	}

	panic(fmt.Errorf("unhandled color type=%d value=%#x", color.colorType(), value))
}

func (color Color) ForegroundAnsiString() string {
	// FIXME: Test this function with all different color types.
	return color.ansiString(true)
}

func (color Color) BackgroundAnsiString() string {
	// FIXME: Test this function with all different color types.
	return color.ansiString(false)
}

func (color Color) String() string {
	switch color.colorType() {
	case colorTypeDefault:
		return "Default color"

	case colorType16:
		return colorNames16[int(color.colorValue())]

	case colorType256:
		if color.colorValue() < 16 {
			return colorNames16[int(color.colorValue())]
		}
		return fmt.Sprintf("#%02x", color.colorValue())

	case colorType24bit:
		return fmt.Sprintf("#%06x", color.colorValue())
	}

	panic(fmt.Errorf("unhandled color type %d", color.colorType()))
}

// Compute 0-255 luminance value.
//
// Works only on colorType24bit colors, check colorType() before calling this
// method.
//
// Ref: https://www.scantips.com/lumin.html
func (color Color) Luminance() (int, error) {
	if color.colorType() != colorType24bit {
		return -1, fmt.Errorf("Color type must be 24 bit, was %d", color.colorType())
	}

	value := color.colorValue()

	red_0_to_255 := float64((value & 0xff0000) >> 16)
	green_0_to_255 := float64((value & 0xff00) >> 8)
	blue_0_to_255 := float64(value & 0xff)

	luminance_0_to_255 := 0.3*red_0_to_255 + 0.59*green_0_to_255 + 0.11*blue_0_to_255

	return int(luminance_0_to_255), nil
}
