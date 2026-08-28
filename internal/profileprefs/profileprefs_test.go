package profileprefs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmartAutofillSettingIsValid(t *testing.T) {
	assert.True(t, SmartAutofillOn.IsValid())
	assert.True(t, SmartAutofillOff.IsValid())
	assert.False(t, SmartAutofillSetting("").IsValid())
	assert.False(t, SmartAutofillSetting("auto").IsValid())
}

func TestDefaultSmartAutofill(t *testing.T) {
	assert.Equal(t, SmartAutofillOn, DefaultSmartAutofill())
}

func TestUserPreferencesDefaults(t *testing.T) {
	prefs := DefaultUserPreferences()
	assert.Equal(t, DefaultTemperatureUnit(), prefs.TemperatureUnit)
	assert.Equal(t, SmartAutofillOn, prefs.SmartAutofill)
}

func TestUserPreferencesWithDefaults(t *testing.T) {
	// Invalid or empty values fall back to the defaults.
	prefs := UserPreferences{
		TemperatureUnit: TemperatureUnit("bogus"),
		SmartAutofill:   SmartAutofillSetting(""),
	}.WithDefaults()
	assert.Equal(t, DefaultTemperatureUnit(), prefs.TemperatureUnit)
	assert.Equal(t, SmartAutofillOn, prefs.SmartAutofill)

	// Valid values are preserved, including an explicit "off".
	prefs = UserPreferences{
		TemperatureUnit: TemperatureUnitFahrenheit,
		SmartAutofill:   SmartAutofillOff,
	}.WithDefaults()
	assert.Equal(t, TemperatureUnitFahrenheit, prefs.TemperatureUnit)
	assert.Equal(t, SmartAutofillOff, prefs.SmartAutofill)
}

func TestUserPreferencesJSON(t *testing.T) {
	// The JSON key is smart_autofill, so a missing key must decode as the
	// zero value (which WithDefaults turns into "on"), not a bool false.
	var prefs UserPreferences
	require.NoError(t, json.Unmarshal([]byte(`{"temperature_unit":"fahrenheit","smart_autofill":"off"}`), &prefs))
	assert.Equal(t, TemperatureUnitFahrenheit, prefs.TemperatureUnit)
	assert.Equal(t, SmartAutofillOff, prefs.SmartAutofill)

	raw, err := json.Marshal(UserPreferences{TemperatureUnit: TemperatureUnitCelsius, SmartAutofill: SmartAutofillOn})
	require.NoError(t, err)
	assert.JSONEq(t, `{"temperature_unit":"celsius","smart_autofill":"on"}`, string(raw))
}
