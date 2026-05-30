package ogcard

import "image/color"

var (
	ColorBg      = color.RGBA{0xFD, 0xFA, 0xF5, 0xFF} // warm cream background
	ColorDark    = color.RGBA{0x2D, 0x1A, 0x0E, 0xFF} // dark brown headings
	ColorBody    = color.RGBA{0x5C, 0x3D, 0x2E, 0xFF} // medium brown body text
	ColorMuted   = color.RGBA{0x8B, 0x6F, 0x5A, 0xFF} // muted brown secondary
	ColorBrand   = color.RGBA{0x4A, 0x2C, 0x2A, 0xFF} // brand dark bar
	ColorWhite   = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	ColorDivider = color.RGBA{0xE0, 0xD5, 0xCA, 0xFF} // divider line
)

const (
	cardWidth  = 1200
	cardHeight = 630
	leftPad    = 60
	stripeW    = 8
	brandBarH  = 60
	brandBarY  = cardHeight - brandBarH
)
