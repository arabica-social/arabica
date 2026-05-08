package ogcard

import (
	"fmt"
	"strings"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// entityStartY computes the starting Y to vertically center contentH within the card.
func entityStartY(contentH int) int {
	y := max((brandBarY-contentH)/2, 30)
	return y
}

// maxTextWidth is the text area width, leaving room for the logo on the right.
var maxTextWidth = contentW - leftPad

// DrawBeanCard generates a 1200x630 OG image for a bean record.
func DrawBeanCard(bean *arabica.Bean) (*Card, error) {
	card, err := newTypedCard(AccentBean, entities.Get(lexicons.RecordTypeBean).Noun)
	if err != nil {
		return nil, err
	}

	x := leftPad

	// Calculate content height for centering
	nameLines := card.CountWrappedLines(bean.Name, maxTextWidth, 44, true, 2)
	nameH := card.LineSpacing(44) * nameLines
	h := nameH
	hasDetails := bean.Origin != "" || bean.RoastLevel != "" || bean.Process != ""
	if hasDetails {
		h += 42
	}
	if bean.Roaster != nil && bean.Roaster.Name != "" {
		h += 42
	}
	if bean.Rating != nil && *bean.Rating > 0 {
		h += 44
	}
	h += 32 // divider
	if bean.Variety != "" {
		h += 34
	}
	if bean.Description != "" {
		h += 10 + 30 // single line
	}

	y := entityStartY(h)

	// Bean name (wrap up to 2 lines)
	y = card.DrawWrappedTextCapped(bean.Name, x, y, maxTextWidth, ColorDark, 44, true, 2)

	// Origin / roast / process
	var details []string
	if bean.Origin != "" {
		details = append(details, bean.Origin)
	}
	if bean.RoastLevel != "" {
		details = append(details, bean.RoastLevel+" Roast")
	}
	if bean.Process != "" {
		details = append(details, bean.Process)
	}
	if len(details) > 0 {
		card.DrawText(truncateLine(card, strings.Join(details, dot), maxTextWidth, 26, false), x, y, ColorBody, 26)
		y += 42
	}

	// Roaster
	if bean.Roaster != nil && bean.Roaster.Name != "" {
		card.DrawText(truncateLine(card, "by "+bean.Roaster.Name, maxTextWidth, 26, false), x, y, ColorBody, 26)
		y += 42
	}

	// Rating bar
	if bean.Rating != nil && *bean.Rating > 0 {
		y += 8
		y = drawRatingBar(card, x, y, *bean.Rating)
	}

	// Divider
	y += 8
	card.DrawRect(x, y, x+maxTextWidth, y+2, ColorDivider)
	y += 22

	// Variety
	if bean.Variety != "" {
		card.DrawBoldText("Variety", x, y, ColorDark, 22)
		card.DrawText(bean.Variety, x+120, y, ColorBody, 22)
		y += 34
	}

	// Description
	if bean.Description != "" {
		y += 10
		card.DrawText(truncateLine(card, bean.Description, maxTextWidth, 24, false), x, y, ColorMuted, 24)
	}

	return card, nil
}

// DrawRoasterCard generates a 1200x630 OG image for a roaster record.
func DrawRoasterCard(roaster *arabica.Roaster) (*Card, error) {
	card, err := newTypedCard(AccentRoaster, entities.Get(lexicons.RecordTypeRoaster).Noun)
	if err != nil {
		return nil, err
	}

	x := leftPad

	nameLines := card.CountWrappedLines(roaster.Name, maxTextWidth, 44, true, 2)
	h := card.LineSpacing(44) * nameLines
	if roaster.Location != "" {
		h += 44
	}
	h += 32 // divider
	if roaster.Website != "" {
		h += 34
	}

	y := entityStartY(h)

	// Roaster name (wrap up to 2 lines)
	y = card.DrawWrappedTextCapped(roaster.Name, x, y, maxTextWidth, ColorDark, 44, true, 2)

	// Location
	if roaster.Location != "" {
		card.DrawText(roaster.Location, x, y, ColorBody, 28)
		y += 44
	}

	// Divider
	y += 8
	card.DrawRect(x, y, x+maxTextWidth, y+2, ColorDivider)
	y += 22

	// Website
	if roaster.Website != "" {
		card.DrawText(roaster.Website, x, y, ColorMuted, 22)
	}

	return card, nil
}

// DrawGrinderCard generates a 1200x630 OG image for a grinder record.
func DrawGrinderCard(grinder *arabica.Grinder) (*Card, error) {
	card, err := newTypedCard(AccentGrinder, entities.Get(lexicons.RecordTypeGrinder).Noun)
	if err != nil {
		return nil, err
	}

	x := leftPad

	nameLines := card.CountWrappedLines(grinder.Name, maxTextWidth, 44, true, 2)
	h := card.LineSpacing(44) * nameLines
	hasDetails := grinder.GrinderType != "" || grinder.BurrType != ""
	if hasDetails {
		h += 44
	}
	h += 32 // divider
	if grinder.Notes != "" {
		h += 10 + 60
	}

	y := entityStartY(h)

	// Grinder name (wrap up to 2 lines)
	y = card.DrawWrappedTextCapped(grinder.Name, x, y, maxTextWidth, ColorDark, 44, true, 2)

	// Type and burr type
	var details []string
	if grinder.GrinderType != "" {
		details = append(details, grinder.GrinderType)
	}
	if grinder.BurrType != "" {
		details = append(details, grinder.BurrType+" burrs")
	}
	if len(details) > 0 {
		card.DrawText(strings.Join(details, dot), x, y, ColorBody, 28)
		y += 44
	}

	// Divider
	y += 8
	card.DrawRect(x, y, x+maxTextWidth, y+2, ColorDivider)
	y += 22

	// Notes
	if grinder.Notes != "" {
		card.DrawText(truncateLine(card, grinder.Notes, maxTextWidth, 24, false), x, y, ColorMuted, 24)
	}

	return card, nil
}

// DrawBrewerCard generates a 1200x630 OG image for a brewer record.
func DrawBrewerCard(brewer *arabica.Brewer) (*Card, error) {
	card, err := newTypedCard(AccentBrewer, entities.Get(lexicons.RecordTypeBrewer).Noun)
	if err != nil {
		return nil, err
	}

	x := leftPad

	nameLines := card.CountWrappedLines(brewer.Name, maxTextWidth, 44, true, 2)
	h := card.LineSpacing(44) * nameLines
	if brewer.BrewerType != "" {
		h += 44
	}
	h += 32 // divider
	if brewer.Description != "" {
		h += 10 + 60
	}

	y := entityStartY(h)

	// Brewer name (wrap up to 2 lines)
	y = card.DrawWrappedTextCapped(brewer.Name, x, y, maxTextWidth, ColorDark, 44, true, 2)

	// Type label
	if brewer.BrewerType != "" {
		typeLabel := brewer.BrewerType
		if label, ok := arabica.BrewerTypeLabels[brewer.BrewerType]; ok {
			typeLabel = label
		}
		card.DrawText(typeLabel, x, y, ColorBody, 28)
		y += 44
	}

	// Divider
	y += 8
	card.DrawRect(x, y, x+maxTextWidth, y+2, ColorDivider)
	y += 22

	// Description
	if brewer.Description != "" {
		card.DrawText(truncateLine(card, brewer.Description, maxTextWidth, 24, false), x, y, ColorMuted, 24)
	}

	return card, nil
}

// DrawRecipeCard generates a 1200x630 OG image for a recipe record.
func DrawRecipeCard(recipe *arabica.Recipe) (*Card, error) {
	var recipeType string
	if recipe.BrewerType != "" {
		recipeType = recipe.BrewerType
	} else {
		recipeType = "brew"
	}

	card, err := newTypedCard(AccentRecipe, recipeType+" recipe")
	if err != nil {
		return nil, err
	}

	x := leftPad

	nameLines := card.CountWrappedLines(recipe.Name, maxTextWidth, 44, true, 2)
	h := card.LineSpacing(44) * nameLines
	hasMethod := recipe.BrewerType != "" || (recipe.BrewerObj != nil && recipe.BrewerObj.Name != "")
	if hasMethod {
		h += 44
	}
	h += 32 // divider
	if recipe.CoffeeAmount > 0 || recipe.WaterAmount > 0 {
		h += 40
	}
	if len(recipe.Pours) > 0 {
		h += 34
	}
	if recipe.Notes != "" {
		h += 10 + 60
	}

	y := entityStartY(h)

	// Recipe name (wrap up to 2 lines)
	y = card.DrawWrappedTextCapped(recipe.Name, x, y, maxTextWidth, ColorDark, 44, true, 2)

	// Brewer type + brewer name
	var methodParts []string
	if recipe.BrewerType != "" {
		if label, ok := arabica.BrewerTypeLabels[recipe.BrewerType]; ok {
			methodParts = append(methodParts, label)
		} else {
			methodParts = append(methodParts, recipe.BrewerType)
		}
	}
	if recipe.BrewerObj != nil && recipe.BrewerObj.Name != "" {
		methodParts = append(methodParts, recipe.BrewerObj.Name)
	}
	if len(methodParts) > 0 {
		card.DrawText(strings.Join(methodParts, dot), x, y, ColorBody, 28)
		y += 44
	}

	// Divider
	y += 8
	card.DrawRect(x, y, x+maxTextWidth, y+2, ColorDivider)
	y += 22

	// Params line: coffee / water / ratio
	var params []string
	if recipe.CoffeeAmount > 0 {
		params = append(params, fmt.Sprintf("%.0fg coffee", recipe.CoffeeAmount))
	}
	if recipe.WaterAmount > 0 {
		params = append(params, fmt.Sprintf("%.0fg water", recipe.WaterAmount))
	}
	if recipe.Ratio > 0 {
		params = append(params, fmt.Sprintf("1:%.1f ratio", recipe.Ratio))
	}
	if len(params) > 0 {
		card.DrawBoldText(strings.Join(params, dot), x, y, ColorDark, 26)
		y += 40
	}

	// Pours
	if len(recipe.Pours) > 0 {
		var pourTexts []string
		for _, p := range recipe.Pours {
			pourTexts = append(pourTexts, fmt.Sprintf("%dg \u00b7 %d:%02d", p.WaterAmount, p.TimeSeconds/60, p.TimeSeconds%60))
		}
		pourText := fmt.Sprintf("%d pours: %s", len(recipe.Pours), strings.Join(pourTexts, ", "))
		card.DrawText(truncate(pourText, 100), x, y, ColorBody, 22)
		y += 34
	}

	// Notes
	if recipe.Notes != "" {
		y += 10
		card.DrawText(truncateLine(card, recipe.Notes, maxTextWidth, 24, false), x, y, ColorMuted, 24)
	}

	return card, nil
}
