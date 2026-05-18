package ogcard

import "image"

// SiteCardOpts configures DrawSiteCard for a specific app/brand.
// Zero values fall back to arabica's defaults so existing callers keep working.
type SiteCardOpts struct {
	// AppName is the lowercase identifier ("arabica", "oolong") used to
	// pick the embedded logo when Logo is nil.
	AppName string
	// Wordmark is the brand text rendered as the title and bottom bar
	// (e.g. "arabica.social", "Oolong"). Defaults to "arabica.social".
	Wordmark string
	// Tagline is the secondary line under the title.
	Tagline string
	// Detail is the third line below the divider.
	Detail string
	// Logo, if set, overrides the per-app embedded logo lookup.
	Logo image.Image
}

func (o SiteCardOpts) wordmark() string {
	if o.Wordmark != "" {
		return o.Wordmark
	}
	return "arabica.social"
}

func (o SiteCardOpts) tagline() string {
	if o.Tagline != "" {
		return o.Tagline
	}
	return "coffee journaling for the open social web"
}

func (o SiteCardOpts) detail() string {
	if o.Detail != "" {
		return o.Detail
	}
	return "track, share, and own your brews"
}

func (o SiteCardOpts) logo() image.Image {
	if o.Logo != nil {
		return o.Logo
	}
	return GetLogoFor(o.AppName)
}

// DrawSiteCard generates a 1200x630 OG image for the root site embed.
// It shows the logo prominently with the site name and tagline.
func DrawSiteCard(opts SiteCardOpts) (*Card, error) {
	card, err := NewCard(cardWidth, cardHeight, ColorBg)
	if err != nil {
		return nil, err
	}

	wordmark := opts.wordmark()

	// Left accent stripe (brand color)
	card.DrawRect(0, 0, stripeW, cardHeight, ColorBrand)

	// Bottom brand bar
	card.DrawRect(0, brandBarY, cardWidth, cardHeight, ColorBrand)
	card.DrawBoldText(wordmark, leftPad, brandBarY+18, ColorWhite, 26)

	// Logo centered in right portion
	if logo := opts.logo(); logo != nil {
		logoSz := 220
		lx := cardWidth - leftPad - logoSz
		ly := (brandBarY - logoSz) / 2
		card.DrawImageScaled(logo, lx, ly, logoSz, logoSz)
	}

	// Text content — vertically centered in left area
	contentH := 54 + 16 + 40 + 24 + 32 + 30 // title + gap + tagline + gap + divider + detail line
	y := (brandBarY - contentH) / 2

	card.DrawBoldText(wordmark, leftPad, y, ColorDark, 48)
	y += 54 + 16

	card.DrawText(opts.tagline(), leftPad, y, ColorBody, 28)
	y += 40 + 24

	// Divider
	card.DrawRect(leftPad, y, leftPad+120, y+2, ColorDivider)
	y += 32

	card.DrawText(opts.detail(), leftPad, y, ColorMuted, 22)

	return card, nil
}
