package models

// Dropdown options for form fields
// These constants eliminate duplication across template files

var (
	// RoastLevels defines the available roast level options for beans
	RoastLevels = []string{
		"Ultra-Light",
		"Light",
		"Medium-Light",
		"Medium",
		"Medium-Dark",
		"Dark",
	}

	// GrinderTypes defines the available grinder type options
	GrinderTypes = []string{
		"Hand",
		"Electric",
		"Portable Electric",
	}

	// BurrTypes defines the available burr type options for grinders
	BurrTypes = []string{
		"Conical",
		"Flat",
	}
)
