package twin

func color256ToRGB(color256 uint8) (r, g, b float64) {
	if color256 < 16 {
		// Standard ANSI colors
		r := float64(standardAnsiColors[color256][0]) / 255.0
		g := float64(standardAnsiColors[color256][1]) / 255.0
		b := float64(standardAnsiColors[color256][2]) / 255.0
		return r, g, b
	}

	if color256 >= 232 {
		// Grayscale. Colors 232-255 map to components 0x08 to 0xee
		gray := float64((color256-232)*0x0a+0x08) / 255.0
		return gray, gray, gray
	}

	// 6x6 color cube
	color0_to_215 := color256 - 16

	components := []uint8{0x00, 0x5f, 0x87, 0xaf, 0xd7, 0xff}
	r = float64(components[(color0_to_215/36)%6]) / 255.0
	g = float64(components[(color0_to_215/6)%6]) / 255.0
	b = float64(components[(color0_to_215/1)%6]) / 255.0
	return r, g, b
}

// This table modified to match the VGA colors documented here:
// https://en.wikipedia.org/wiki/ANSI_escape_code#3-bit_and_4-bit
var standardAnsiColors = [16][3]uint8{
	{0, 0, 0},       // Black
	{170, 0, 0},     // Red
	{0, 170, 0},     // Green
	{170, 85, 0},    // Yellow
	{0, 0, 170},     // Blue
	{170, 0, 170},   // Magenta
	{0, 170, 170},   // Cyan
	{170, 170, 170}, // White
	{85, 85, 85},    // Bright Black
	{255, 85, 85},   // Bright Red
	{85, 255, 85},   // Bright Green
	{255, 255, 85},  // Bright Yellow
	{85, 85, 255},   // Bright Blue
	{255, 85, 255},  // Bright Magenta
	{85, 255, 255},  // Bright Cyan
	{255, 255, 255}, // Bright White
}
