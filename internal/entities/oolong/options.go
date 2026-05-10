package oolong

// Dropdown option lists for oolong form fields. Re-exports of the
// knownValues slices in models_*.go.

var (
	// Categories are the broad tea classifications.
	Categories = CategoryKnownValues

	// BrewerStyles are vessel styles (gaiwan, kyusu, ...).
	BrewerStyles = BrewerStyleKnownValues

	// BrewStyles are the four brewing styles (gongfu, matcha, longSteep, milkTea).
	BrewStyles = StyleKnownValues

	// MatchaPreparations enumerate matcha prep variants.
	MatchaPreparations = MatchaPreparationKnownValues

	// IngredientUnits are units of measure for milk-tea ingredients.
	IngredientUnits = IngredientUnitKnownValues

	// ProcessingSteps are tea processing chain steps.
	ProcessingSteps = ProcessingKnownValues
)
