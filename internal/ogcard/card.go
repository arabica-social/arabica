package ogcard

import (
	"bytes"
	_ "embed"
	"image"
	"image/color"
	"image/draw"
	_ "image/jpeg"
	"image/png"
	"io"
	"strings"
	"sync"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed fonts/patricks-iosevka-regular.ttf
var regularTTF []byte

//go:embed fonts/patricks-iosevka-semibold.ttf
var semiboldTTF []byte

//go:embed arabica-logo.png
var logoPNG []byte

var logoImage image.Image
var logoOnce sync.Once

// Card represents a drawable image canvas for generating OG card images.
type Card struct {
	Img    *image.RGBA
	Width  int
	Height int
	Margin int
}

var (
	parsedRegular *opentype.Font
	parsedBold    *opentype.Font
	parseOnce     sync.Once
	parseErr      error

	faceCache sync.Map // map[faceKey]font.Face
)

type faceKey struct {
	bold bool
	size float64
}

func initFonts() {
	parseOnce.Do(func() {
		parsedRegular, parseErr = opentype.Parse(regularTTF)
		if parseErr != nil {
			return
		}
		parsedBold, parseErr = opentype.Parse(semiboldTTF)
	})
}

func getFace(bold bool, sizePt float64) (font.Face, error) {
	key := faceKey{bold, sizePt}
	if v, ok := faceCache.Load(key); ok {
		return v.(font.Face), nil
	}
	f := parsedRegular
	if bold {
		f = parsedBold
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    sizePt,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	faceCache.Store(key, face)
	return face, nil
}

// NewCard creates a new card filled with the given background color.
func NewCard(width, height int, bg color.Color) (*Card, error) {
	initFonts()
	if parseErr != nil {
		return nil, parseErr
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{bg}, image.Point{}, draw.Src)
	return &Card{Img: img, Width: width, Height: height}, nil
}

// SetMargin sets inner margins used by Split.
func (c *Card) SetMargin(m int) { c.Margin = m }

// Split divides the card. If vertical, splits left/right; otherwise top/bottom.
// The first returned card gets the given percentage.
func (c *Card) Split(vertical bool, percentage int) (*Card, *Card) {
	b := c.Img.Bounds()
	inner := image.Rect(b.Min.X+c.Margin, b.Min.Y+c.Margin, b.Max.X-c.Margin, b.Max.Y-c.Margin)
	if vertical {
		mid := inner.Min.X + inner.Dx()*percentage/100
		l := c.Img.SubImage(image.Rect(inner.Min.X, inner.Min.Y, mid, inner.Max.Y)).(*image.RGBA)
		r := c.Img.SubImage(image.Rect(mid, inner.Min.Y, inner.Max.X, inner.Max.Y)).(*image.RGBA)
		return &Card{Img: l, Width: l.Bounds().Dx(), Height: l.Bounds().Dy()},
			&Card{Img: r, Width: r.Bounds().Dx(), Height: r.Bounds().Dy()}
	}
	mid := inner.Min.Y + inner.Dy()*percentage/100
	t := c.Img.SubImage(image.Rect(inner.Min.X, inner.Min.Y, inner.Max.X, mid)).(*image.RGBA)
	bot := c.Img.SubImage(image.Rect(inner.Min.X, mid, inner.Max.X, inner.Max.Y)).(*image.RGBA)
	return &Card{Img: t, Width: t.Bounds().Dx(), Height: t.Bounds().Dy()},
		&Card{Img: bot, Width: bot.Bounds().Dx(), Height: bot.Bounds().Dy()}
}

// DrawText draws regular text. x,y is the top-left of the text bounding box.
// Returns the advance width in pixels.
func (c *Card) DrawText(text string, x, y int, clr color.Color, sizePt float64) int {
	return c.drawString(text, x, y, clr, sizePt, false)
}

// DrawBoldText draws bold text. x,y is the top-left. Returns advance width.
func (c *Card) DrawBoldText(text string, x, y int, clr color.Color, sizePt float64) int {
	return c.drawString(text, x, y, clr, sizePt, true)
}

func (c *Card) drawString(text string, x, y int, clr color.Color, sizePt float64, bold bool) int {
	fc, err := getFace(bold, sizePt)
	if err != nil {
		return 0
	}
	baseline := y + fc.Metrics().Ascent.Ceil()
	d := &font.Drawer{
		Dst:  c.Img,
		Src:  image.NewUniform(clr),
		Face: fc,
		Dot:  fixed.P(x, baseline),
	}
	start := d.Dot.X
	d.DrawString(text)
	return (d.Dot.X - start).Ceil()
}

// MeasureText returns the pixel width of text at the given size.
func (c *Card) MeasureText(text string, sizePt float64, bold bool) int {
	fc, err := getFace(bold, sizePt)
	if err != nil {
		return 0
	}
	return font.MeasureString(fc, text).Ceil()
}

// FontHeight returns the line height for the given point size.
func (c *Card) FontHeight(sizePt float64) int {
	fc, err := getFace(false, sizePt)
	if err != nil {
		return int(sizePt)
	}
	m := fc.Metrics()
	return (m.Ascent + m.Descent).Ceil()
}

// DrawWrappedText draws word-wrapped text within maxWidth pixels.
// Returns the Y position after the last line.
func (c *Card) DrawWrappedText(text string, x, y, maxWidth int, clr color.Color, sizePt float64, bold bool) int {
	fc, err := getFace(bold, sizePt)
	if err != nil {
		return y
	}
	m := fc.Metrics()
	lineHeight := (m.Ascent + m.Descent).Ceil()
	lineSpacing := lineHeight + lineHeight/4

	words := strings.Fields(text)
	if len(words) == 0 {
		return y
	}

	currentLine := ""
	currentY := y

	for _, word := range words {
		proposed := currentLine
		if proposed != "" {
			proposed += " "
		}
		proposed += word

		if font.MeasureString(fc, proposed).Ceil() > maxWidth && currentLine != "" {
			c.drawString(currentLine, x, currentY, clr, sizePt, bold)
			currentY += lineSpacing
			currentLine = word
		} else {
			currentLine = proposed
		}
	}
	if currentLine != "" {
		c.drawString(currentLine, x, currentY, clr, sizePt, bold)
		currentY += lineSpacing
	}
	return currentY
}

// DrawWrappedTextCapped draws word-wrapped text within maxWidth, capped at maxLines.
// If the text overflows, the last line is truncated with "…".
// Returns the Y position after the last line.
func (c *Card) DrawWrappedTextCapped(text string, x, y, maxWidth int, clr color.Color, sizePt float64, bold bool, maxLines int) int {
	fc, err := getFace(bold, sizePt)
	if err != nil {
		return y
	}
	m := fc.Metrics()
	lineHeight := (m.Ascent + m.Descent).Ceil()
	lineSpacing := lineHeight + lineHeight/4

	words := strings.Fields(text)
	if len(words) == 0 {
		return y
	}

	var lines []string
	currentLine := ""
	for _, word := range words {
		proposed := currentLine
		if proposed != "" {
			proposed += " "
		}
		proposed += word
		if font.MeasureString(fc, proposed).Ceil() > maxWidth && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = proposed
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	if len(lines) > maxLines {
		lines = lines[:maxLines]
		last := []rune(lines[maxLines-1])
		for len(last) > 0 && font.MeasureString(fc, string(last)+"…").Ceil() > maxWidth {
			last = last[:len(last)-1]
		}
		lines[maxLines-1] = string(last) + "…"
	}

	currentY := y
	for _, line := range lines {
		c.drawString(line, x, currentY, clr, sizePt, bold)
		currentY += lineSpacing
	}
	return currentY
}

// DrawRect fills a rectangle with the given color.
func (c *Card) DrawRect(x1, y1, x2, y2 int, clr color.Color) {
	draw.Draw(c.Img, image.Rect(x1, y1, x2, y2), &image.Uniform{clr}, image.Point{}, draw.Src)
}

// DrawRoundedRect fills a rounded rectangle.
func (c *Card) DrawRoundedRect(x, y, w, h, radius int, clr color.RGBA) {
	bounds := c.Img.Bounds()
	for py := y; py < y+h; py++ {
		for px := x; px < x+w; px++ {
			dx, dy := 0, 0
			switch {
			case px < x+radius && py < y+radius:
				dx, dy = x+radius-px, y+radius-py
			case px >= x+w-radius && py < y+radius:
				dx, dy = px-(x+w-radius-1), y+radius-py
			case px < x+radius && py >= y+h-radius:
				dx, dy = x+radius-px, py-(y+h-radius-1)
			case px >= x+w-radius && py >= y+h-radius:
				dx, dy = px-(x+w-radius-1), py-(y+h-radius-1)
			}
			inCorner := dx > 0 || dy > 0
			withinRadius := dx*dx+dy*dy <= radius*radius
			if (!inCorner || withinRadius) && px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
				c.Img.Set(px, py, clr)
			}
		}
	}
}

// DrawImageScaled draws an image scaled to fit within the given rectangle,
// centered and maintaining aspect ratio.
func (c *Card) DrawImageScaled(img image.Image, x, y, w, h int) {
	srcB := img.Bounds()
	srcW, srcH := float64(srcB.Dx()), float64(srcB.Dy())
	targetW, targetH := float64(w), float64(h)

	scale := targetW / srcW
	if s := targetH / srcH; s < scale {
		scale = s
	}

	newW := int(srcW * scale)
	newH := int(srcH * scale)
	offX := x + (w-newW)/2
	offY := y + (h-newH)/2

	dst := image.Rect(offX, offY, offX+newW, offY+newH)
	xdraw.CatmullRom.Scale(c.Img, dst, img, srcB, xdraw.Over, nil)
}

// GetLogo returns the embedded arabica logo image.
func GetLogo() image.Image {
	logoOnce.Do(func() {
		logoImage, _ = png.Decode(bytes.NewReader(logoPNG))
	})
	return logoImage
}

// EncodePNG writes the card image as PNG.
func (c *Card) EncodePNG(w io.Writer) error {
	return png.Encode(w, c.Img)
}
