package ai

import (
	"strings"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

// Recipe preparation is one proposal: ambiguous identity must not leak its
// measurements into the draft before the user chooses the source recipe.
func recipeDetail(field string) bool {
	switch field {
	case "recipe_rkey", "recipe_owner_did", "coffee_amount", "water_amount", "temperature", "time_seconds", "grind_size", "method", "pours":
		return true
	}
	return strings.HasPrefix(field, "espresso_params.") || strings.HasPrefix(field, "pourover_params.")
}

func (c catalog) brewerCategory(rkey string) string {
	for _, item := range c {
		if item.Kind == "brewer" && item.rkey == rkey {
			return arabica.NormalizeBrewerType(item.BrewerType)
		}
	}
	return ""
}

func (c catalog) brewerType(rkey string) string {
	for _, item := range c {
		if item.Kind == "brewer" && item.rkey == rkey {
			return item.BrewerType
		}
	}
	return ""
}

// enrichRecipe fills absent values from the PDS recipe, never overwriting an
// explicit proposal. A linked own brewer is authoritative relationship data,
// not a fuzzy model guess. Existing recipe selections do not reset manual edits.
func (c catalog) enrichRecipe(patch, current map[string]any) (string, error) {
	rkey, ok := patch["recipe_rkey"].(string)
	if !ok || rkey == "" || current["recipe_rkey"] == rkey {
		return "", nil
	}
	baseline, sourceType, err := c.recipePatch(rkey)
	if err != nil {
		return "", err
	}
	sourceCategory := arabica.NormalizeBrewerType(sourceType)
	if sourceCategory == "" {
		for _, item := range c {
			if item.Kind == "recipe" && item.rkey == rkey {
				sourceCategory = arabica.NormalizeBrewerType(item.BrewerType)
				break
			}
		}
	}
	selected, explicit := patch["brewer_rkey"]
	if !explicit {
		selected = baseline["brewer_rkey"]
	}
	if selected == nil {
		selected = current["brewer_rkey"]
	}
	selectedKey, _ := selected.(string)
	compatiblePours := sourceCategory == arabica.BrewerTypePourover && c.brewerCategory(selectedKey) == arabica.BrewerTypePourover
	for key, value := range baseline {
		if key == "pours" && !compatiblePours {
			continue
		}
		if _, exists := patch[key]; !exists {
			patch[key] = value
		}
	}
	linked, _ := baseline["brewer_rkey"].(string)
	if linked != "" && patch["brewer_rkey"] == linked {
		return linked, nil
	}
	return "", nil
}

func takeRecipeProposal(patch map[string]any) map[string]any {
	pick := map[string]any{}
	_, hasGrind := patch["grind_size"]
	for field, value := range patch {
		if recipeDetail(field) || field == "brewer_rkey" || (hasGrind && field == "grinder_rkey") {
			pick[field] = value
			delete(patch, field)
		}
	}
	return pick
}

// Only a roaster request gets open-roaster alternatives. Exact bean names still
// follow the ordinary duplicate-label rule. Counts include nonretained records.
func (c catalog) openRoasterAlternatives(rkey, message string) ([]Choice, int, string) {
	roaster := ""
	total := 0
	for _, item := range c {
		if item.Kind == "bean" && item.rkey == rkey && item.RoasterLabel != "" && exactMention(message, item.RoasterLabel) {
			roaster, total = item.RoasterLabel, item.OpenRoasterMatches
			break
		}
	}
	if roaster == "" {
		return nil, 0, ""
	}
	picks := []Choice{}
	for _, item := range c {
		if item.Kind == "bean" && normalized(item.RoasterLabel) == normalized(roaster) && item.Closed != nil && !*item.Closed {
			patch := map[string]any{"bean_rkey": item.rkey}
			picks = append(picks, Choice{Label: c.deriveLabel(patch), Patch: patch})
		}
	}
	return picks, total, roaster
}

func replyNote(reply, note string) string {
	if len([]rune(reply))+len([]rune(note))+2 > 2000 {
		return note
	}
	return reply + "\n\n" + note
}

// filterContradictedRecipes removes only a recipe proposal disproved by an
// explicit identifying pour count. Source preparation and brewer stay atomic;
// unrelated bean/grinder choices survive. Validate relationships before pruning.
func (c catalog) filterContradictedRecipes(patch map[string]any, choices []Choice, current map[string]any) (map[string]any, []Choice, error) {
	strip := func(proposal map[string]any) (map[string]any, bool, error) {
		for _, item := range c {
			if item.Kind != "recipe" || item.rkey != proposal["recipe_rkey"] || !item.pourCountMismatch {
				continue
			}
			clean := make(map[string]any, len(proposal))
			for key, value := range proposal {
				clean[key] = value
			}
			if _, err := c.enrichRecipe(clean, current); err != nil {
				return nil, false, err
			}
			takeRecipeProposal(clean)
			return clean, true, nil
		}
		return proposal, false, nil
	}
	cleanPatch, _, err := strip(patch)
	if err != nil {
		return nil, nil, err
	}
	result := make([]Choice, 0, len(choices))
	for _, choice := range choices {
		clean, removed, err := strip(choice.Patch)
		if err != nil {
			return nil, nil, err
		}
		if len(clean) == 0 {
			continue
		}
		if removed {
			choice.Patch = clean
			choice.Label = c.deriveLabel(clean)
			choice.BrewerType = ""
		}
		result = append(result, choice)
	}
	return cleanPatch, result, nil
}
