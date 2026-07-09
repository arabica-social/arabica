package handlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/profileprefs"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// SettingsResponseJSON is the JSON envelope returned by GET /api/settings.
type SettingsResponseJSON struct {
	ProfileStatsVisibility profileprefs.ProfileStatsVisibility `json:"profile_stats_visibility"`
	UserPreferences        profileprefs.UserPreferences        `json:"user_preferences"`
	BlueskyProfile         BlueskyProfileJSON                  `json:"bluesky_profile"`
}

// BlueskyProfileJSON carries the Bluesky profile sync state for the settings page.
type BlueskyProfileJSON struct {
	HasScopes      bool   `json:"has_scopes"`
	DisplayName    string `json:"display_name,omitempty"`
	AvatarURL      string `json:"avatar_url,omitempty"`
	LoadError      string `json:"load_error,omitempty"`
	NeedsAuthAgain bool   `json:"needs_auth_again,omitempty"`
}

// SettingsSavedResponseJSON is the JSON response for settings mutation endpoints.
type SettingsSavedResponseJSON struct {
	Saved bool `json:"saved"`
}

// HandleSettingsJSON returns the user's settings as JSON for the SvelteKit SPA.
func (h *Handler) HandleSettingsJSON(w http.ResponseWriter, r *http.Request) {
	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}
	sessionID, _ := atpmiddleware.GetSessionID(r.Context())

	var statsVis profileprefs.ProfileStatsVisibility
	prefs := profileprefs.DefaultUserPreferences()
	if h.feedIndex != nil {
		statsVis = h.feedIndex.GetProfileStatsVisibility(r.Context(), didStr)
		prefs = h.feedIndex.GetUserPreferences(r.Context(), didStr)
	} else {
		statsVis = profileprefs.DefaultProfileStatsVisibility()
	}

	bskyForm := h.loadBlueskyProfileForm(r.Context(), didStr, sessionID)

	WriteJSON(w, SettingsResponseJSON{
		ProfileStatsVisibility: statsVis,
		UserPreferences:        prefs,
		BlueskyProfile: BlueskyProfileJSON{
			HasScopes:      bskyForm.HasScopes,
			DisplayName:    bskyForm.DisplayName,
			AvatarURL:      bskyForm.AvatarURL,
			LoadError:      bskyForm.LoadError,
			NeedsAuthAgain: bskyForm.NeedsAuthAgain,
		},
	}, "settings")
}

// HandleSettingsPreferencesJSON saves user preferences and returns JSON.
func (h *Handler) HandleSettingsPreferencesJSON(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid form")
		return
	}

	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	prefs := profileprefs.UserPreferences{
		TemperatureUnit: profileprefs.TemperatureUnit(r.FormValue("temperature_unit")),
	}.WithDefaults()

	if h.feedIndex != nil {
		if err := h.feedIndex.SetUserPreferences(r.Context(), didStr, prefs); err != nil {
			log.Error().Err(err).Msg("Failed to save user preferences")
			WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to save preferences")
			return
		}
	}

	WriteJSON(w, SettingsSavedResponseJSON{Saved: true}, "settings-preferences")
}

// HandleSettingsProfileVisibilityJSON saves profile visibility and returns JSON.
func (h *Handler) HandleSettingsProfileVisibilityJSON(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		WriteJSONError(w, http.StatusBadRequest, "invalid_request", "Invalid form")
		return
	}

	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
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
			WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to save settings")
			return
		}
	}

	WriteJSON(w, SettingsSavedResponseJSON{Saved: true}, "settings-visibility")
}
