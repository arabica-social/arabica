package oolong

import "errors"

// Field length limits, mirrored from lexicon JSONs.
const (
	MaxNameLength         = 200
	MaxLocationLength     = 200
	MaxAddressLength      = 500
	MaxWebsiteLength      = 500
	MaxDescriptionLength  = 1000
	MaxNotesLength        = 2000
	MaxOriginLength       = 200
	MaxCultivarLength     = 200
	MaxFarmLength         = 200
	MaxSubStyleLength     = 200
	MaxCategoryLength     = 50
	MaxStepLength         = 50
	MaxStepDetailLength   = 300
	MaxBrewerStyleLength  = 50
	MaxMaterialLength     = 200
	MaxMenuItemLength     = 200
	MaxStyleLength        = 50
	MaxTastingNotesLength = 2000
	MaxSteepNotesLength   = 500
	MaxPreparationLength  = 300
	MaxIngredientName     = 200
	MaxIngredientUnit     = 20
	MaxIngredientNotes    = 200
	MaxWhiskTypeLength    = 200
	MaxCommentText        = 1000
	MaxCommentGraphemes   = 300
)

// Validation errors. Mirrors arabica's error sentinel pattern.
var (
	ErrNameRequired     = errors.New("name is required")
	ErrNameTooLong      = errors.New("name is too long")
	ErrFieldTooLong     = errors.New("field value is too long")
	ErrLocationTooLong  = errors.New("location is too long")
	ErrWebsiteTooLong   = errors.New("website is too long")
	ErrDescTooLong      = errors.New("description is too long")
	ErrRatingOutOfRange = errors.New("rating must be between 1 and 10")
	ErrTeaRefRequired   = errors.New("teaRef is required")
	ErrCafeRefRequired  = errors.New("cafeRef is required")
	ErrSubjectRequired  = errors.New("subject is required")
	ErrTextRequired     = errors.New("text is required")
	ErrTextTooLong      = errors.New("text is too long")
	ErrParentInvalid    = errors.New("parent_uri and parent_cid must be provided together")
	ErrStyleRequired    = errors.New("style is required")
	ErrStyleInvalid     = errors.New("style is not a known value")
	ErrCategoryInvalid  = errors.New("category is not a known value")
	ErrIngredientNoName = errors.New("ingredient name is required")
)

// toFloat64 extracts a numeric value from an interface{} that may be int or
// float64. JSON decoding produces float64; in-memory maps may contain int.
// Mirrors the helper of the same name in internal/entities/arabica/records.go.
func toFloat64(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
