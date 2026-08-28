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

// SmartAutofillSetting is the user's preference for automatically filling
// empty brew details on the new-brew form from their own recent brews.
type SmartAutofillSetting string

const (
	SmartAutofillOn  SmartAutofillSetting = "on"
	SmartAutofillOff SmartAutofillSetting = "off"
)

func (s SmartAutofillSetting) IsValid() bool {
	switch s {
	case SmartAutofillOn, SmartAutofillOff:
		return true
	}
	return false
}

// DefaultSmartAutofill returns the default smart autofill preference. The
// feature helps most users, so it defaults to on.
func DefaultSmartAutofill() SmartAutofillSetting {
	return SmartAutofillOn
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
	TemperatureUnit TemperatureUnit      `json:"temperature_unit"`
	SmartAutofill   SmartAutofillSetting `json:"smart_autofill"`
}

func DefaultUserPreferences() UserPreferences {
	return UserPreferences{
		TemperatureUnit: DefaultTemperatureUnit(),
		SmartAutofill:   DefaultSmartAutofill(),
	}
}

func (p UserPreferences) WithDefaults() UserPreferences {
	if !p.TemperatureUnit.IsValid() {
		p.TemperatureUnit = DefaultTemperatureUnit()
	}
	if !p.SmartAutofill.IsValid() {
		p.SmartAutofill = DefaultSmartAutofill()
	}
	return p
}
