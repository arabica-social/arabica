// Package lexicons defines types for Arabica's AT Protocol lexicon schemas.
package lexicons

// RecordType represents the type of a record in the feed.
// Use these constants instead of magic strings for type safety.
type RecordType string

const (
	RecordTypeBean    RecordType = "bean"
	RecordTypeBrew    RecordType = "brew"
	RecordTypeBrewer  RecordType = "brewer"
	RecordTypeGrinder RecordType = "grinder"
	RecordTypeLike    RecordType = "like"
	RecordTypeRecipe  RecordType = "recipe"
	RecordTypeRoaster RecordType = "roaster"
)

const (
	RecordTypeOolongTea     RecordType = "oolong-tea"
	RecordTypeOolongBrew    RecordType = "oolong-brew"
	RecordTypeOolongVessel  RecordType = "oolong-vessel"
	RecordTypeOolongInfuser RecordType = "oolong-infuser"
	RecordTypeOolongVendor  RecordType = "oolong-vendor"
	// Cafe and Drink are defined but deferred for the v1 launch — their
	// descriptors are not registered. Keep the constants so feed/firehose
	// code that switches on record type still has a stable name.
	RecordTypeOolongCafe  RecordType = "oolong-cafe"
	RecordTypeOolongDrink RecordType = "oolong-drink"
)

// String returns the string representation of the RecordType.
func (r RecordType) String() string {
	return string(r)
}

// ParseRecordType converts a string to a RecordType if valid, returns empty string if not.
func ParseRecordType(s string) RecordType {
	switch RecordType(s) {
	case RecordTypeBean, RecordTypeBrew, RecordTypeBrewer, RecordTypeGrinder, RecordTypeRecipe, RecordTypeRoaster:
		return RecordType(s)
	case RecordTypeOolongTea, RecordTypeOolongBrew, RecordTypeOolongVessel, RecordTypeOolongInfuser, RecordTypeOolongVendor, RecordTypeOolongCafe, RecordTypeOolongDrink:
		return RecordType(s)
	default:
		return ""
	}
}

// DisplayName returns a human-readable name for the RecordType.
func (r RecordType) DisplayName() string {
	switch r {
	case RecordTypeBean:
		return "Bean"
	case RecordTypeBrew:
		return "Brew"
	case RecordTypeBrewer:
		return "Brewer"
	case RecordTypeGrinder:
		return "Grinder"
	case RecordTypeLike:
		return "Like"
	case RecordTypeRecipe:
		return "Recipe"
	case RecordTypeRoaster:
		return "Roaster"
	case RecordTypeOolongTea:
		return "Tea"
	case RecordTypeOolongBrew:
		return "Tea Brew"
	case RecordTypeOolongVessel:
		return "Vessel"
	case RecordTypeOolongInfuser:
		return "Infuser"
	case RecordTypeOolongVendor:
		return "Tea Vendor"
	case RecordTypeOolongCafe:
		return "Tea Cafe"
	case RecordTypeOolongDrink:
		return "Tea Drink"
	default:
		return string(r)
	}
}
