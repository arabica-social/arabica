package arabica

import (
	"errors"
	"strings"
	"time"

	"tangled.org/arabica.social/arabica/internal/notifications"
	"tangled.org/arabica.social/arabica/internal/profileprefs"
	"tangled.org/arabica.social/arabica/internal/social"
)

// Field length limits for validation
const (
	MaxNameLength         = 200
	MaxLocationLength     = 200
	MaxWebsiteLength      = 500
	MaxLinkLength         = 500
	MaxDescriptionLength  = 2000
	MaxNotesLength        = 2000
	MaxOriginLength       = 200
	MaxRoastLevelLength   = 100
	MaxVarietyLength      = 200
	MaxProcessLength      = 100
	MaxMethodLength       = 100
	MaxGrindSizeLength    = 100
	MaxTastingNotesLength = 2000
	MaxGrinderTypeLength  = 50
	MaxBurrTypeLength     = 50
	MaxBrewerTypeLength   = 100
)

const MaxCommentLength = social.MaxCommentLength

type Visibility = profileprefs.Visibility

const (
	VisibilityPublic  = profileprefs.VisibilityPublic
	VisibilityPrivate = profileprefs.VisibilityPrivate
)

type ProfileStatsVisibility = profileprefs.ProfileStatsVisibility

// DefaultProfileStatsVisibility returns the default visibility (all public).
func DefaultProfileStatsVisibility() ProfileStatsVisibility {
	return profileprefs.DefaultProfileStatsVisibility()
}

// Brewer type categories (knownValues from lexicon)
const (
	BrewerTypePourover  = "pourover"
	BrewerTypeEspresso  = "espresso"
	BrewerTypeImmersion = "immersion"
	BrewerTypeMokaPot   = "mokapot"
	BrewerTypeColdBrew  = "coldbrew"
	BrewerTypeCupping   = "cupping"
	BrewerTypeOther     = "other"
)

// BrewerTypeLabels maps canonical brewer type values to display labels
var BrewerTypeLabels = map[string]string{
	BrewerTypePourover:  "Pour-over",
	BrewerTypeEspresso:  "Espresso",
	BrewerTypeImmersion: "Immersion",
	BrewerTypeMokaPot:   "Moka Pot",
	BrewerTypeColdBrew:  "Cold Brew",
	BrewerTypeCupping:   "Cupping",
	BrewerTypeOther:     "Other",
}

// BrewerTypeKnownValues is the ordered list for form dropdowns
var BrewerTypeKnownValues = []string{
	BrewerTypePourover,
	BrewerTypeEspresso,
	BrewerTypeImmersion,
	BrewerTypeMokaPot,
	BrewerTypeColdBrew,
	BrewerTypeCupping,
	BrewerTypeOther,
}

// NormalizeBrewerType maps freeform brewer type strings to canonical values.
// Returns the input unchanged if no mapping is found (preserves unknown values).
func NormalizeBrewerType(raw string) string {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case lower == "pourover" || lower == "pour-over" || lower == "pour over" || lower == "dripper":
		return BrewerTypePourover
	case lower == "espresso" || lower == "espresso machine" || lower == "lever espresso" || lower == "lever espresso machine":
		return BrewerTypeEspresso
	case lower == "immersion" || lower == "french press" || lower == "aeropress" || lower == "siphon" || lower == "clever" || lower == "clever dripper":
		return BrewerTypeImmersion
	case lower == "mokapot" || lower == "moka pot" || lower == "moka" || lower == "bialetti":
		return BrewerTypeMokaPot
	case lower == "coldbrew" || lower == "cold brew" || lower == "cold drip":
		return BrewerTypeColdBrew
	case lower == "cupping":
		return BrewerTypeCupping
	case lower == "other":
		return BrewerTypeOther
	default:
		return raw // preserve unknown values
	}
}

// Validation errors
var (
	ErrNameRequired     = errors.New("name is required")
	ErrNameTooLong      = errors.New("name is too long")
	ErrLocationTooLong  = errors.New("location is too long")
	ErrWebsiteTooLong   = errors.New("website is too long")
	ErrLinkTooLong      = errors.New("link is too long")
	ErrDescTooLong      = errors.New("description is too long")
	ErrNotesTooLong     = errors.New("notes is too long")
	ErrOriginTooLong    = errors.New("origin is too long")
	ErrFieldTooLong     = errors.New("field value is too long")
	ErrRatingOutOfRange = errors.New("rating must be between 1 and 10")
	ErrCommentRequired  = social.ErrCommentRequired
	ErrCommentTooLong   = social.ErrCommentTooLong
)

// TODO: maybe add a "rating" field that can be updated when a bag is closed
type Bean struct {
	RKey        string    `json:"rkey"` // Record key (AT Protocol or stringified ID for SQLite)
	Name        string    `json:"name"`
	Origin      string    `json:"origin"`
	Variety     string    `json:"variety"`
	RoastLevel  string    `json:"roast_level"`
	Process     string    `json:"process"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	RoasterRKey string    `json:"roaster_rkey"`     // AT Protocol reference
	Rating      *int      `json:"rating,omitempty"` // User rating (1-10), nil means unrated
	Closed      bool      `json:"closed"`           // Whether the bag is closed/finished
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`

	// Joined data for display
	Roaster *Roaster `json:"roaster,omitempty"`
}

type Roaster struct {
	RKey      string    `json:"rkey"` // Record key
	Name      string    `json:"name"`
	Location  string    `json:"location"`
	Website   string    `json:"website"`
	SourceRef string    `json:"source_ref,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Grinder struct {
	RKey        string    `json:"rkey"` // Record key
	Name        string    `json:"name"`
	GrinderType string    `json:"grinder_type"` // Hand, Electric, Portable Electric
	BurrType    string    `json:"burr_type"`    // Conical, Flat, Blade, or empty
	Notes       string    `json:"notes"`
	Link        string    `json:"link"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Brewer struct {
	RKey        string    `json:"rkey"` // Record key
	Name        string    `json:"name"`
	BrewerType  string    `json:"brewer_type"`
	Description string    `json:"description"`
	Link        string    `json:"link"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Pour struct {
	PourNumber  int       `json:"pour_number"`
	WaterAmount int       `json:"water_amount"`
	TimeSeconds int       `json:"time_seconds"`
	CreatedAt   time.Time `json:"created_at"`
}

// EspressoParams holds espresso-specific brewing parameters
type EspressoParams struct {
	YieldWeight        float64 `json:"yield_weight"`         // Output weight in grams
	Pressure           float64 `json:"pressure"`             // Pressure in bar
	PreInfusionSeconds int     `json:"pre_infusion_seconds"` // Pre-infusion time
}

// PouroverParams holds pour-over-specific brewing parameters
type PouroverParams struct {
	BloomWater      int    `json:"bloom_water"`      // Bloom water in grams
	BloomSeconds    int    `json:"bloom_seconds"`    // Bloom wait time in seconds
	DrawdownSeconds int    `json:"drawdown_seconds"` // Drawdown time in seconds
	BypassWater     int    `json:"bypass_water"`     // Bypass water in grams
	Filter          string `json:"filter"`           // Filter type (e.g. "paper", "metal", "cloth")
}

type Recipe struct {
	RKey         string    `json:"rkey"`
	Name         string    `json:"name"`
	BrewerRKey   string    `json:"brewer_rkey"`
	BrewerType   string    `json:"brewer_type"`
	CoffeeAmount float64   `json:"coffee_amount"`
	WaterAmount  float64   `json:"water_amount"`
	Notes        string    `json:"notes"`
	SourceRef    string    `json:"source_ref,omitempty"`
	CreatedAt    time.Time `json:"created_at"`

	// Joined data for display
	BrewerObj *Brewer `json:"brewer_obj,omitempty"`
	Pours     []*Pour `json:"pours,omitempty"`

	// Computed fields (populated by Interpolate or handler)
	Ratio         float64 `json:"ratio,omitempty"`          // water:coffee ratio (e.g. 16.7 for 1:16.7)
	AuthorDID     string  `json:"author_did,omitempty"`     // DID of the recipe creator
	AuthorHandle  string  `json:"author_handle,omitempty"`  // handle of the creator
	AuthorAvatar  string  `json:"author_avatar,omitempty"`  // avatar URL of the creator
	AuthorDisplay string  `json:"author_display,omitempty"` // display name of the creator

	// Source/fork provenance (populated by handler for explore/view)
	SourceAuthorHandle  string `json:"source_author_handle,omitempty"`
	SourceAuthorAvatar  string `json:"source_author_avatar,omitempty"`
	SourceAuthorDisplay string `json:"source_author_display,omitempty"`

	// Social stats (populated by handler for explore)
	ForkCount     int      `json:"fork_count,omitempty"`
	BrewCount     int      `json:"brew_count,omitempty"`
	ForkerAvatars []string `json:"forker_avatars,omitempty"` // up to N forker profile pics
}

// Interpolate fills in computed/derived fields from existing data.
// - BrewerType from BrewerObj if not set
// - WaterAmount from sum of pours if not set
// - Ratio from water/coffee amounts
func (r *Recipe) Interpolate() {
	// Derive brewer type from joined brewer object if missing
	if r.BrewerType == "" && r.BrewerObj != nil && r.BrewerObj.BrewerType != "" {
		r.BrewerType = r.BrewerObj.BrewerType
	}
	// Derive water amount from pours if missing
	if r.WaterAmount == 0 && len(r.Pours) > 0 {
		var total int
		for _, p := range r.Pours {
			total += p.WaterAmount
		}
		r.WaterAmount = float64(total)
	}
	// Compute ratio
	if r.CoffeeAmount > 0 && r.WaterAmount > 0 {
		r.Ratio = r.WaterAmount / r.CoffeeAmount
	}
}

type Brew struct {
	RKey         string    `json:"rkey"` // Record key
	BeanRKey     string    `json:"bean_rkey"`
	RecipeRKey   string    `json:"recipe_rkey"`
	Method       string    `json:"method,omitempty"`
	Temperature  float64   `json:"temperature"`
	WaterAmount  int       `json:"water_amount"`
	CoffeeAmount int       `json:"coffee_amount"`
	TimeSeconds  int       `json:"time_seconds"`
	GrindSize    string    `json:"grind_size"`
	GrinderRKey  string    `json:"grinder_rkey"`
	BrewerRKey   string    `json:"brewer_rkey"`
	TastingNotes string    `json:"tasting_notes"`
	Rating       int       `json:"rating"`
	CreatedAt    time.Time `json:"created_at"`

	// Method-specific parameters
	EspressoParams *EspressoParams `json:"espresso_params,omitempty"`
	PouroverParams *PouroverParams `json:"pourover_params,omitempty"`

	// Joined data for display
	Bean       *Bean    `json:"bean,omitempty"`
	RecipeObj  *Recipe  `json:"recipe_obj,omitempty"`
	GrinderObj *Grinder `json:"grinder_obj,omitempty"`
	BrewerObj  *Brewer  `json:"brewer_obj,omitempty"`
	Pours      []*Pour  `json:"pours,omitempty"`
}

type CreateBrewRequest struct {
	BeanRKey       string           `json:"bean_rkey"`
	RecipeRKey     string           `json:"recipe_rkey"`
	RecipeOwnerDID string           `json:"recipe_owner_did"` // DID of the recipe owner (may differ from brew author)
	Method         string           `json:"method"`
	Temperature    float64          `json:"temperature"`
	WaterAmount    int              `json:"water_amount"`
	CoffeeAmount   int              `json:"coffee_amount"`
	TimeSeconds    int              `json:"time_seconds"`
	GrindSize      string           `json:"grind_size"`
	GrinderRKey    string           `json:"grinder_rkey"`
	BrewerRKey     string           `json:"brewer_rkey"`
	TastingNotes   string           `json:"tasting_notes"`
	Rating         int              `json:"rating"`
	Pours          []CreatePourData `json:"pours"`
	EspressoParams *EspressoParams  `json:"espresso_params,omitempty"`
	PouroverParams *PouroverParams  `json:"pourover_params,omitempty"`
}

type CreatePourData struct {
	WaterAmount int `json:"water_amount"`
	TimeSeconds int `json:"time_seconds"`
}

type CreateBeanRequest struct {
	Name        string `json:"name"`
	Origin      string `json:"origin"`
	Variety     string `json:"variety"`
	RoastLevel  string `json:"roast_level"`
	Process     string `json:"process"`
	Description string `json:"description"`
	Link        string `json:"link"`
	RoasterRKey string `json:"roaster_rkey"`
	Rating      *int   `json:"rating,omitempty"`
	Closed      bool   `json:"closed"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type CreateRoasterRequest struct {
	Name      string `json:"name"`
	Location  string `json:"location"`
	Website   string `json:"website"`
	SourceRef string `json:"source_ref,omitempty"`
}

type CreateGrinderRequest struct {
	Name        string `json:"name"`
	GrinderType string `json:"grinder_type"`
	BurrType    string `json:"burr_type"`
	Notes       string `json:"notes"`
	Link        string `json:"link"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type CreateBrewerRequest struct {
	Name        string `json:"name"`
	BrewerType  string `json:"brewer_type"`
	Description string `json:"description"`
	Link        string `json:"link"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type CreateRecipeRequest struct {
	Name         string           `json:"name"`
	BrewerRKey   string           `json:"brewer_rkey"`
	BrewerType   string           `json:"brewer_type"`
	CoffeeAmount float64          `json:"coffee_amount"`
	WaterAmount  float64          `json:"water_amount"`
	Notes        string           `json:"notes"`
	SourceRef    string           `json:"source_ref,omitempty"`
	Pours        []CreatePourData `json:"pours"`
}

type UpdateRecipeRequest struct {
	Name         string           `json:"name"`
	BrewerRKey   string           `json:"brewer_rkey"`
	BrewerType   string           `json:"brewer_type"`
	CoffeeAmount float64          `json:"coffee_amount"`
	WaterAmount  float64          `json:"water_amount"`
	Notes        string           `json:"notes"`
	Pours        []CreatePourData `json:"pours"`
}

type UpdateBeanRequest struct {
	Name        string `json:"name"`
	Origin      string `json:"origin"`
	Variety     string `json:"variety"`
	RoastLevel  string `json:"roast_level"`
	Process     string `json:"process"`
	Description string `json:"description"`
	Link        string `json:"link"`
	RoasterRKey string `json:"roaster_rkey"`
	Rating      *int   `json:"rating,omitempty"`
	Closed      bool   `json:"closed"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateRoasterRequest struct {
	Name      string `json:"name"`
	Location  string `json:"location"`
	Website   string `json:"website"`
	SourceRef string `json:"source_ref,omitempty"`
}

type UpdateGrinderRequest struct {
	Name        string `json:"name"`
	GrinderType string `json:"grinder_type"`
	BurrType    string `json:"burr_type"`
	Notes       string `json:"notes"`
	Link        string `json:"link"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateBrewerRequest struct {
	Name        string `json:"name"`
	BrewerType  string `json:"brewer_type"`
	Description string `json:"description"`
	Link        string `json:"link"`
	SourceRef   string `json:"source_ref,omitempty"`
}

// IsIncomplete returns true if the bean is missing key fields beyond name/origin.
func (b *Bean) IsIncomplete() bool {
	return b.RoasterRKey == "" || b.RoastLevel == ""
}

// MissingFields returns a human-readable list of missing fields.
func (b *Bean) MissingFields() []string {
	var missing []string
	if b.RoasterRKey == "" {
		missing = append(missing, "roaster")
	}
	if b.RoastLevel == "" {
		missing = append(missing, "roast level")
	}
	return missing
}

// IsIncomplete returns true if the grinder is missing its type.
func (g *Grinder) IsIncomplete() bool {
	return g.GrinderType == ""
}

// MissingFields returns a human-readable list of missing fields.
func (g *Grinder) MissingFields() []string {
	var missing []string
	if g.GrinderType == "" {
		missing = append(missing, "grinder type")
	}
	return missing
}

// IsIncomplete returns true if the brewer is missing its type.
func (b *Brewer) IsIncomplete() bool {
	return b.BrewerType == ""
}

// MissingFields returns a human-readable list of missing fields.
func (b *Brewer) MissingFields() []string {
	var missing []string
	if b.BrewerType == "" {
		missing = append(missing, "brewer type")
	}
	return missing
}

// Like represents a like on an Arabica record.
type Like = social.Like

// CreateLikeRequest contains the data needed to create a like.
type CreateLikeRequest = social.CreateLikeRequest

// Comment represents a comment on an Arabica record.
type Comment = social.Comment

// CreateCommentRequest contains the data needed to create a comment.
type CreateCommentRequest = social.CreateCommentRequest

// Validate checks that all fields are within acceptable limits
func (r *CreateBeanRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Origin) > MaxOriginLength {
		return ErrOriginTooLong
	}
	if len(r.Variety) > MaxVarietyLength {
		return ErrFieldTooLong
	}
	if len(r.RoastLevel) > MaxRoastLevelLength {
		return ErrFieldTooLong
	}
	if len(r.Process) > MaxProcessLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	if len(r.Link) > MaxLinkLength {
		return ErrLinkTooLong
	}
	if r.Rating != nil && (*r.Rating < 1 || *r.Rating > 10) {
		return ErrRatingOutOfRange
	}
	return nil
}

// Validate checks that all fields are within acceptable limits
func (r *UpdateBeanRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Origin) > MaxOriginLength {
		return ErrOriginTooLong
	}
	if len(r.Variety) > MaxVarietyLength {
		return ErrFieldTooLong
	}
	if len(r.RoastLevel) > MaxRoastLevelLength {
		return ErrFieldTooLong
	}
	if len(r.Process) > MaxProcessLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	if len(r.Link) > MaxLinkLength {
		return ErrLinkTooLong
	}
	if r.Rating != nil && (*r.Rating < 1 || *r.Rating > 10) {
		return ErrRatingOutOfRange
	}
	return nil
}

// Validate checks that all fields are within acceptable limits
func (r *CreateRoasterRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Location) > MaxLocationLength {
		return ErrLocationTooLong
	}
	if len(r.Website) > MaxWebsiteLength {
		return ErrWebsiteTooLong
	}
	return nil
}

// Validate checks that all fields are within acceptable limits
func (r *UpdateRoasterRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Location) > MaxLocationLength {
		return ErrLocationTooLong
	}
	if len(r.Website) > MaxWebsiteLength {
		return ErrWebsiteTooLong
	}
	return nil
}

// Validate checks that all fields are within acceptable limits
func (r *CreateGrinderRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.GrinderType) > MaxGrinderTypeLength {
		return ErrFieldTooLong
	}
	if len(r.BurrType) > MaxBurrTypeLength {
		return ErrFieldTooLong
	}
	if len(r.Notes) > MaxNotesLength {
		return ErrNotesTooLong
	}
	if len(r.Link) > MaxLinkLength {
		return ErrLinkTooLong
	}
	return nil
}

// Validate checks that all fields are within acceptable limits
func (r *UpdateGrinderRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.GrinderType) > MaxGrinderTypeLength {
		return ErrFieldTooLong
	}
	if len(r.BurrType) > MaxBurrTypeLength {
		return ErrFieldTooLong
	}
	if len(r.Notes) > MaxNotesLength {
		return ErrNotesTooLong
	}
	if len(r.Link) > MaxLinkLength {
		return ErrLinkTooLong
	}
	return nil
}

// Validate checks that all fields are within acceptable limits
func (r *CreateBrewerRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.BrewerType) > MaxBrewerTypeLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	if len(r.Link) > MaxLinkLength {
		return ErrLinkTooLong
	}
	return nil
}

// Validate checks that all fields are within acceptable limits
func (r *CreateRecipeRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.BrewerType) > MaxBrewerTypeLength {
		return ErrFieldTooLong
	}
	if len(r.Notes) > MaxNotesLength {
		return ErrNotesTooLong
	}
	return nil
}

// Validate checks that all fields are within acceptable limits
func (r *UpdateRecipeRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.BrewerType) > MaxBrewerTypeLength {
		return ErrFieldTooLong
	}
	if len(r.Notes) > MaxNotesLength {
		return ErrNotesTooLong
	}
	return nil
}

// Validate checks that all string fields are within acceptable limits
func (r *CreateBrewRequest) Validate() error {
	if len(r.Method) > MaxMethodLength {
		return ErrFieldTooLong
	}
	if len(r.GrindSize) > MaxGrindSizeLength {
		return ErrFieldTooLong
	}
	if len(r.TastingNotes) > MaxTastingNotesLength {
		return ErrFieldTooLong
	}
	return nil
}

// Validate checks that all fields are within acceptable limits
func (r *UpdateBrewerRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.BrewerType) > MaxBrewerTypeLength {
		return ErrFieldTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	if len(r.Link) > MaxLinkLength {
		return ErrLinkTooLong
	}
	return nil
}

// NotificationType represents the type of notification
type NotificationType = notifications.Type

const (
	NotificationLike         = notifications.Like
	NotificationComment      = notifications.Comment
	NotificationCommentReply = notifications.CommentReply
)

// Notification represents a notification for a user
type Notification = notifications.Notification

// Report represents a user-submitted content report
// TODO: Store reports in database (BoltDB or SQLite) for moderation review
type Report struct {
	ID          string    `json:"id"`
	SubjectURI  string    `json:"subject_uri"`  // AT-URI of the reported content
	SubjectCID  string    `json:"subject_cid"`  // CID of the reported content
	Reason      string    `json:"reason"`       // spam, inappropriate, other
	ReporterDID string    `json:"reporter_did"` // DID of reporter, or "anonymous"
	ReporterIP  string    `json:"reporter_ip"`  // IP address for rate limiting
	CreatedAt   time.Time `json:"created_at"`
	Status      string    `json:"status"` // pending, reviewed, dismissed, actioned
}
