package oolong

// Dropdown option lists for oolong form fields. Re-exports of the
// knownValues slices in models_*.go.

var (
	// Categories are the broad tea classifications.
	Categories = CategoryKnownValues

	// VesselStyles are vessel form-factors (teapot, mug, jar, ...).
	VesselStyles = VesselStyleKnownValues

	// InfuserStyles are infuser form-factors (basket, ball, sock, ...).
	InfuserStyles = InfuserStyleKnownValues

	// BrewStyles are the brewing styles in scope for v1 (longSteep, coldBrew).
	BrewStyles = StyleKnownValues

	// InfusionMethods enumerate how the leaf was contained in a brew.
	InfusionMethods = InfusionMethodKnownValues
)
