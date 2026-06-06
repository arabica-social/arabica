package handlers

import (
	"context"
	"net/http"

	"tangled.org/arabica.social/arabica/internal/profileprefs"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/web/pages"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// About page.
func (h *Handler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	data, _, _ := h.LayoutDataFromRequest(r, "About")

	render := h.staticPages.About
	if render == nil {
		render = func(ctx context.Context, w http.ResponseWriter, data *components.LayoutData) error {
			return pages.About(data).Render(ctx, w)
		}
	}
	if err := render(r.Context(), w, data); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render about page")
	}
}

// Terms of Service page.
func (h *Handler) HandleTerms(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.LayoutDataFromRequest(r, "Terms of Service")

	render := h.staticPages.Terms
	if render == nil {
		render = func(ctx context.Context, w http.ResponseWriter, data *components.LayoutData) error {
			return pages.Terms(data).Render(ctx, w)
		}
	}
	if err := render(r.Context(), w, layoutData); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render terms page")
	}
}

func (h *Handler) HandleATProto(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.LayoutDataFromRequest(r, "AT Protocol")

	if err := pages.ATProto(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render AT Protocol page")
	}
}

// Settings page
func (h *Handler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	data, _, isAuthenticated := h.LayoutDataFromRequest(r, "Settings")
	if !isAuthenticated {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	didStr, _ := atpmiddleware.GetDID(r.Context())
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

	if err := pages.Settings(data, pages.SettingsProps{
		ProfileStatsVisibility: statsVis,
		UserPreferences:        prefs,
		BlueskyProfile: pages.BlueskyProfileSettings{
			HasScopes:      bskyForm.HasScopes,
			DisplayName:    bskyForm.DisplayName,
			AvatarURL:      bskyForm.AvatarURL,
			LoadError:      bskyForm.LoadError,
			NeedsAuthAgain: bskyForm.NeedsAuthAgain,
		},
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render settings page")
	}
}

func (h *Handler) HandleSettingsPreferences(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	prefs := profileprefs.UserPreferences{
		TemperatureUnit: profileprefs.TemperatureUnit(r.FormValue("temperature_unit")),
	}.WithDefaults()

	if h.feedIndex != nil {
		if err := h.feedIndex.SetUserPreferences(r.Context(), didStr, prefs); err != nil {
			log.Error().Err(err).Msg("Failed to save user preferences")
			http.Error(w, "Failed to save preferences", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span class="text-sm text-green-700 dark:text-green-400">Saved</span>`))
}

func (h *Handler) HandleSettingsProfileVisibility(w http.ResponseWriter, r *http.Request) {
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

	// Validate — fall back to public for unrecognized values
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

	// Return a success indicator for HTMX
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(`<span class="text-sm text-green-700 dark:text-green-400">Saved</span>`))
}

// HandleNotFound renders the 404 page
func (h *Handler) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.LayoutDataFromRequest(r, "Page Not Found")

	w.WriteHeader(http.StatusNotFound)
	if err := pages.NotFound(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render 404 page")
	}
}
