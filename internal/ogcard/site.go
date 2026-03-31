package ogcard

// DrawSiteCard generates a 1200x630 OG image for the root site embed.
// It shows the logo prominently with the site name and tagline.
func DrawSiteCard() (*Card, error) {
	card, err := NewCard(cardWidth, cardHeight, ColorBg)
	if err != nil {
		return nil, err
	}

	// Left accent stripe (brand color)
	card.DrawRect(0, 0, stripeW, cardHeight, ColorBrand)

	// Bottom brand bar
	card.DrawRect(0, brandBarY, cardWidth, cardHeight, ColorBrand)
	card.DrawBoldText("arabica.social", leftPad, brandBarY+18, ColorWhite, 26)

	// Logo centered in right portion
	if logo := GetLogo(); logo != nil {
		logoSz := 220
		lx := 820
		ly := (brandBarY - logoSz) / 2
		card.DrawImageScaled(logo, lx, ly, logoSz, logoSz)
	}

	// Text content — vertically centered in left area
	contentH := 54 + 16 + 40 + 24 + 32 + 30 // title + gap + tagline + gap + divider + detail line
	y := (brandBarY - contentH) / 2

	card.DrawBoldText("arabica.social", leftPad, y, ColorDark, 48)
	y += 54 + 16

	card.DrawText("coffee brew tracking on the AT Protocol", leftPad, y, ColorBody, 28)
	y += 40 + 24

	// Divider
	card.DrawRect(leftPad, y, leftPad+120, y+2, ColorDivider)
	y += 32

	card.DrawText("your data, your PDS, your coffee", leftPad, y, ColorMuted, 22)

	return card, nil
}
