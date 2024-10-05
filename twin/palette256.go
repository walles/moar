package twin

func color256ToRGB(color256 uint8) (r, g, b uint8) {
	if color256 < 16 {
		// Standard ANSI colors
		r := standardAnsiColors[color256][0]
		g := standardAnsiColors[color256][1]
		b := standardAnsiColors[color256][2]
		return r, g, b
	}

	if color256 >= 232 {
		// Grayscale. Colors 232-255 map to components 0x08 to 0xee
		gray := (color256-232)*0x0a + 0x08
		return gray, gray, gray
	}

	// 6x6 color cube
	color0to215 := color256 - 16

	components := []uint8{0x00, 0x5f, 0x87, 0xaf, 0xd7, 0xff}
	r = components[(color0to215/36)%6]
	g = components[(color0to215/6)%6]
	b = components[(color0to215/1)%6]
	return r, g, b
}

// Source, the VGA color palette from here:
// https://en.wikipedia.org/wiki/ANSI_escape_code#3-bit_and_4-bit
var standardAnsiColors = [16][3]uint8{
	{0x00, 0x00, 0x00}, // Black
	{0xAA, 0x00, 0x00}, // Red
	{0x00, 0xAA, 0x00}, // Green
	{0xAA, 0x55, 0x00}, // Yellow
	{0x00, 0x00, 0xAA}, // Blue
	{0xAA, 0x00, 0xAA}, // Magenta
	{0x00, 0xAA, 0xAA}, // Cyan
	{0xAA, 0xAA, 0xAA}, // White

	{0x55, 0x55, 0x55}, // Bright Black
	{0xff, 0x55, 0x55}, // Bright Red
	{0x55, 0xff, 0x55}, // Bright Green
	{0xff, 0xff, 0x55}, // Bright Yellow
	{0x55, 0x55, 0xff}, // Bright Blue
	{0xff, 0x55, 0xff}, // Bright Magenta
	{0x55, 0xff, 0xff}, // Bright Cyan
	{0xff, 0xff, 0xff}, // Bright White
}
