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
	RecordTypeRoaster RecordType = "roaster"
)

// String returns the string representation of the RecordType.
func (r RecordType) String() string {
	return string(r)
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
	case RecordTypeRoaster:
		return "Roaster"
	default:
		return string(r)
	}
}
