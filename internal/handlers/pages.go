package handlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/profileprefs"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// preferencesFromForm applies only submitted fields; absent fields retain the
// stored value while explicitly invalid values use the default.
func preferencesFromForm(r *http.Request, p profileprefs.UserPreferences) profileprefs.UserPreferences {
	if r.Form.Has("temperature_unit") {
		p.TemperatureUnit = profileprefs.TemperatureUnit(r.FormValue("temperature_unit"))
	}
	if r.Form.Has("smart_autofill") {
		p.SmartAutofill = profileprefs.SmartAutofillSetting(r.FormValue("smart_autofill"))
	}
	if r.Form.Has("brew_form_mode") {
		p.BrewFormMode = profileprefs.BrewFormMode(r.FormValue("brew_form_mode"))
	}
	return p.WithDefaults()
}

// HandleSettingsPreferences saves the user's brewing preferences.
func (h *Handler) HandleSettingsPreferences(w http.ResponseWriter, r *http.Request) {
	if WantsJSON(r) {
		h.HandleSettingsPreferencesJSON(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if h.feedIndex != nil {
		if err := h.feedIndex.UpdateUserPreferences(r.Context(), didStr, func(p profileprefs.UserPreferences) profileprefs.UserPreferences {
			return preferencesFromForm(r, p)
		}); err != nil {
			log.Error().Err(err).Msg("Failed to save user preferences")
			http.Error(w, "Failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span class="text-sm text-green-700 dark:text-green-400">Saved</span>`))
}

func (h *Handler) HandleSettingsProfileVisibility(w http.ResponseWriter, r *http.Request) {
	if WantsJSON(r) {
		h.HandleSettingsProfileVisibilityJSON(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	settings := profileprefs.ProfileStatsVisibility{
		BeanAvgRating:    profileprefs.Visibility(r.FormValue("bean_avg_rating")),
		RoasterAvgRating: profileprefs.Visibility(r.FormValue("roaster_avg_rating")),
	}

	if !settings.BeanAvgRating.IsValid() {
		settings.BeanAvgRating = profileprefs.VisibilityPublic
	}
	if !settings.RoasterAvgRating.IsValid() {
		settings.RoasterAvgRating = profileprefs.VisibilityPublic
	}

	if h.feedIndex != nil {
		if err := h.feedIndex.SetProfileStatsVisibility(r.Context(), didStr, settings); err != nil {
			log.Error().Err(err).Msg("Failed to save profile visibility settings")
			http.Error(w, "Failed to save settings", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span class="text-sm text-green-700 dark:text-green-400">Saved</span>`))
}

// HandleNotFound renders a plain 404 response.
func (h *Handler) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Not Found"))
}
