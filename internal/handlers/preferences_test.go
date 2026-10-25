package handlers

import (
	"net/http/httptest"
	"net/url"
	"testing"

	"tangled.org/arabica.social/arabica/internal/profileprefs"
)

func TestPreferencesPartialForm(t *testing.T) {
	saved := profileprefs.UserPreferences{TemperatureUnit: profileprefs.TemperatureUnitFahrenheit, SmartAutofill: profileprefs.SmartAutofillOff, BrewFormMode: profileprefs.BrewFormDetailed}
	for _, tc := range []struct {
		name string
		form url.Values
		mode profileprefs.BrewFormMode
	}{
		{"omitted", url.Values{}, profileprefs.BrewFormDetailed},
		{"quick", url.Values{"brew_form_mode": {"quick"}}, profileprefs.BrewFormQuick},
		{"invalid", url.Values{"brew_form_mode": {"other"}}, profileprefs.BrewFormQuick},
		{"empty", url.Values{"brew_form_mode": {""}}, profileprefs.BrewFormQuick},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/", nil)
			r.Form = tc.form
			got := preferencesFromForm(r, saved)
			if got.BrewFormMode != tc.mode || got.TemperatureUnit != saved.TemperatureUnit || got.SmartAutofill != saved.SmartAutofill {
				t.Fatalf("unexpected preferences: %+v", got)
			}
		})
	}
	if got := (profileprefs.UserPreferences{}).WithDefaults(); got.BrewFormMode != profileprefs.BrewFormQuick {
		t.Fatalf("missing mode: %+v", got)
	}
}
