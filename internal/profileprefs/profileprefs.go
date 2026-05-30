package profileprefs

type Visibility string

const (
	VisibilityPublic  Visibility = "public"
	VisibilityPrivate Visibility = "private"
)

// IsValid returns true if the visibility value is recognized.
func (v Visibility) IsValid() bool {
	switch v {
	case VisibilityPublic, VisibilityPrivate:
		return true
	}
	return false
}

// ProfileStatsVisibility controls visibility for aggregate profile stats.
type ProfileStatsVisibility struct {
	BeanAvgRating    Visibility `json:"bean_avg_rating"`
	RoasterAvgRating Visibility `json:"roaster_avg_rating"`
}

// DefaultProfileStatsVisibility returns the default visibility.
func DefaultProfileStatsVisibility() ProfileStatsVisibility {
	return ProfileStatsVisibility{
		BeanAvgRating:    VisibilityPublic,
		RoasterAvgRating: VisibilityPublic,
	}
}
