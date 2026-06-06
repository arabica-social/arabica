package profileprefs

// TemperatureUnit is the user's preferred display unit for brew temperatures.
type TemperatureUnit string

const (
	TemperatureUnitRecorded   TemperatureUnit = "recorded"
	TemperatureUnitCelsius    TemperatureUnit = "celsius"
	TemperatureUnitFahrenheit TemperatureUnit = "fahrenheit"
)

func (u TemperatureUnit) IsValid() bool {
	switch u {
	case TemperatureUnitRecorded, TemperatureUnitCelsius, TemperatureUnitFahrenheit:
		return true
	}
	return false
}

// DefaultTemperatureUnit returns the default display unit. Stored brew values
// are unchanged; this only controls presentation and future form defaults.
func DefaultTemperatureUnit() TemperatureUnit {
	return TemperatureUnitRecorded
}

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

// UserPreferences groups account-level preferences that should follow the user
// DID across devices and sessions. Device-local preferences (currently theme)
// intentionally stay outside this struct.
type UserPreferences struct {
	TemperatureUnit TemperatureUnit `json:"temperature_unit"`
}

func DefaultUserPreferences() UserPreferences {
	return UserPreferences{TemperatureUnit: DefaultTemperatureUnit()}
}

func (p UserPreferences) WithDefaults() UserPreferences {
	if !p.TemperatureUnit.IsValid() {
		p.TemperatureUnit = DefaultTemperatureUnit()
	}
	return p
}
