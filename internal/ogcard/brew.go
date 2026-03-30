package ogcard

import (
	"fmt"
	"image/color"
	"strings"

	"arabica/internal/models"
)

// Arabica color palette (warm coffee tones)
var (
	ColorBg       = color.RGBA{0xFD, 0xFA, 0xF5, 0xFF} // warm cream background
	ColorDark     = color.RGBA{0x2D, 0x1A, 0x0E, 0xFF} // dark brown headings
	ColorBody     = color.RGBA{0x5C, 0x3D, 0x2E, 0xFF} // medium brown body text
	ColorMuted    = color.RGBA{0x8B, 0x6F, 0x5A, 0xFF} // muted brown secondary
	ColorBrand    = color.RGBA{0x4A, 0x2C, 0x2A, 0xFF} // brand dark bar
	ColorWhite    = color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	ColorGold     = color.RGBA{0xD9, 0x7A, 0x06, 0xFF} // rating fill
	ColorBarEmpty = color.RGBA{0xE8, 0xDD, 0xD3, 0xFF} // rating bar background
	ColorDivider  = color.RGBA{0xE0, 0xD5, 0xCA, 0xFF} // divider line
	ColorBrandSub = color.RGBA{0xBB, 0xA0, 0x90, 0xFF} // brand subtitle text

	// Type accent colors (match CSS --type-* variables)
	AccentBrew    = color.RGBA{0x6B, 0x44, 0x23, 0xFF} // #6b4423
	AccentBean    = color.RGBA{0xB4, 0x53, 0x09, 0xFF} // #b45309
	AccentRecipe  = color.RGBA{0x7F, 0x55, 0x39, 0xFF} // #7f5539
	AccentRoaster = color.RGBA{0x92, 0x40, 0x0E, 0xFF} // #92400e
	AccentGrinder = color.RGBA{0x6B, 0x44, 0x23, 0xFF} // #6b4423
	AccentBrewer  = color.RGBA{0x6B, 0x44, 0x23, 0xFF} // #6b4423
)

const (
	cardWidth  = 1200
	cardHeight = 630
	leftPad    = 60
	stripeW    = 8
	brandBarH  = 60
	brandBarY  = cardHeight - brandBarH // 570
	contentW   = 850                    // width available for text (left of logo)
	logoSize   = 160
	logoX      = 980
	dot        = "  \u00b7  " // spaced middle dot separator
)

// newTypedCard creates a 1200x630 card with accent stripe, brand bar, and logo.
func newTypedCard(accent color.RGBA, typeLabel string) (*Card, error) {
	card, err := NewCard(cardWidth, cardHeight, ColorBg)
	if err != nil {
		return nil, err
	}
	// Left accent stripe
	card.DrawRect(0, 0, stripeW, cardHeight, accent)

	// Logo on right side
	if logo := GetLogo(); logo != nil {
		logoY := (brandBarY - logoSize) / 2
		card.DrawImageScaled(logo, logoX, logoY, logoSize, logoSize)
	}

	// Bottom brand bar
	card.DrawRect(0, brandBarY, cardWidth, cardHeight, ColorBrand)
	w := card.DrawBoldText("arabica.social", leftPad, brandBarY+18, ColorWhite, 26)
	card.DrawText(typeLabel, leftPad+w+16, brandBarY+22, ColorBrandSub, 18)
	return card, nil
}

// drawRatingBar renders a horizontal rating bar with filled/empty portions.
func drawRatingBar(card *Card, x, y, rating int) int {
	barWidth := 240
	barHeight := 16
	radius := 8
	card.DrawRoundedRect(x, y, barWidth, barHeight, radius, ColorBarEmpty)
	fillWidth := barWidth * rating / 10
	if fillWidth > 0 {
		card.DrawRoundedRect(x, y, fillWidth, barHeight, radius, ColorGold)
	}
	card.DrawBoldText(fmt.Sprintf("%d / 10", rating), x+barWidth+20, y-4, ColorGold, 24)
	return y + 36
}

// truncate shortens s to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

// truncateLine shortens s to fit within maxWidth pixels at sizePt, appending "..." if truncated.
func truncateLine(card *Card, s string, maxWidth int, sizePt float64, bold bool) string {
	if card.MeasureText(s, sizePt, bold) <= maxWidth {
		return s
	}
	for len(s) > 0 {
		candidate := s + "..."
		if card.MeasureText(candidate, sizePt, bold) <= maxWidth {
			return candidate
		}
		// trim one rune at a time
		s = s[:len(s)-1]
	}
	return "..."
}

// brewContentHeight calculates the total height of all brew content lines
// so we can vertically center the content.
func brewContentHeight(card *Card, brew *models.Brew) int {
	h := 0

	// Bean name (1 or 2 lines)
	beanNameForHeight := "Coffee Brew"
	if brew.Bean != nil && brew.Bean.Name != "" {
		beanNameForHeight = brew.Bean.Name
	}
	if card.MeasureText(beanNameForHeight, 44, true) > contentW-leftPad {
		h += 116
	} else {
		h += 58
	}

	// Roaster line
	if brew.Bean != nil && brew.Bean.Roaster != nil && brew.Bean.Roaster.Name != "" {
		h += 38
	}

	// Details line (origin / roast level)
	if brew.Bean != nil && (brew.Bean.Origin != "" || brew.Bean.RoastLevel != "") {
		h += 38
	}

	// Rating bar
	if brew.Rating > 0 {
		h += 8 + 36
	}

	// Divider
	h += 8 + 2 + 22

	// // Method line
	// if (brew.BrewerObj != nil && (brew.BrewerObj.BrewerType != "" || brew.BrewerObj.Name != "")) || brew.Method != "" {
	// 	h += 40
	// }

	// Params line
	if brew.CoffeeAmount > 0 || brew.WaterAmount > 0 || brew.Temperature > 0 || brew.TimeSeconds > 0 {
		h += 38
	}

	// // Grinder info
	// if brew.GrinderObj != nil && brew.GrinderObj.Name != "" {
	// 	h += 34
	// }

	// Tasting notes (2 lines)
	if brew.TastingNotes != "" {
		lh := card.FontHeight(24)
		h += 10 + 2*(lh+lh/4)
	}

	return h
}

// DrawBrewCard generates a 1200x630 OG image for a brew record.
func DrawBrewCard(brew *models.Brew) (*Card, error) {
	card, err := newTypedCard(AccentBrew, "brew")
	if err != nil {
		return nil, err
	}

	x := leftPad
	maxTextW := contentW - leftPad

	// Calculate vertical centering
	contentH := brewContentHeight(card, brew)
	availableH := brandBarY // total space above brand bar
	y := max((availableH-contentH)/2, 30)

	// Bean name (wraps to second line if needed)
	beanName := "Coffee Brew"
	if brew.Bean != nil && brew.Bean.Name != "" {
		beanName = brew.Bean.Name
	}
	y = card.DrawWrappedText(beanName, x, y, maxTextW, ColorDark, 44, true)
	y += 6

	// Roaster line
	if brew.Bean != nil && brew.Bean.Roaster != nil && brew.Bean.Roaster.Name != "" {
		card.DrawText("by "+brew.Bean.Roaster.Name, x, y, ColorBody, 26)
		y += 38
	}

	// Details line: origin / roast level
	var details []string
	if brew.Bean != nil {
		if brew.Bean.Origin != "" {
			details = append(details, brew.Bean.Origin)
		}
		if brew.Bean.RoastLevel != "" {
			details = append(details, brew.Bean.RoastLevel+" Roast")
		}
	}
	if len(details) > 0 {
		card.DrawText(strings.Join(details, dot), x, y, ColorBody, 24)
		y += 38
	}

	// Rating bar
	if brew.Rating > 0 {
		y += 8
		y = drawRatingBar(card, x, y, brew.Rating)
	}

	// Divider
	y += 8
	card.DrawRect(x, y, x+maxTextW, y+2, ColorDivider)
	y += 22

	// // Method line (commented out — too cramped)
	// var methodParts []string
	// if brew.BrewerObj != nil {
	// 	if brew.BrewerObj.BrewerType != "" {
	// 		if label, ok := models.BrewerTypeLabels[brew.BrewerObj.BrewerType]; ok {
	// 			methodParts = append(methodParts, label)
	// 		}
	// 	}
	// 	if brew.BrewerObj.Name != "" {
	// 		methodParts = append(methodParts, brew.BrewerObj.Name)
	// 	}
	// } else if brew.Method != "" {
	// 	methodParts = append(methodParts, brew.Method)
	// }
	// if len(methodParts) > 0 {
	// 	card.DrawBoldText(strings.Join(methodParts, dot), x, y, ColorDark, 28)
	// 	y += 40
	// }

	// Params line
	var params []string
	if brew.CoffeeAmount > 0 {
		params = append(params, fmt.Sprintf("%dg coffee", brew.CoffeeAmount))
	}
	if brew.WaterAmount > 0 {
		waterText := fmt.Sprintf("%dg water", brew.WaterAmount)
		if len(brew.Pours) > 0 {
			waterText += fmt.Sprintf(" (%d pours)", len(brew.Pours))
		}
		params = append(params, waterText)
	}
	if brew.Temperature > 0 {
		params = append(params, fmt.Sprintf("%.0f\u00b0C", brew.Temperature))
	}
	if brew.TimeSeconds > 0 {
		params = append(params, fmt.Sprintf("%d:%02d", brew.TimeSeconds/60, brew.TimeSeconds%60))
	}
	if len(params) > 0 {
		card.DrawText(strings.Join(params, dot), x, y, ColorBody, 24)
		y += 38
	}

	// // Grinder info with grind size in parens
	// if brew.GrinderObj != nil && brew.GrinderObj.Name != "" {
	// 	grinderText := "Ground with " + brew.GrinderObj.Name
	// 	if brew.GrindSize != "" {
	// 		grinderText += " (" + brew.GrindSize + ")"
	// 	}
	// 	card.DrawText(grinderText, x, y, ColorMuted, 22)
	// 	y += 34
	// } else if brew.GrindSize != "" {
	// 	card.DrawText("Grind: "+brew.GrindSize, x, y, ColorMuted, 22)
	// 	y += 34
	// }

	// Tasting notes (2 lines with ellipsis)
	if brew.TastingNotes != "" {
		y += 10
		noteText := "\"" + brew.TastingNotes + "\""
		card.DrawWrappedTextCapped(noteText, x, y, maxTextW, ColorMuted, 24, false, 2)
	}

	return card, nil
}
