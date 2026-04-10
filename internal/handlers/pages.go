package handlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/models"
	"tangled.org/arabica.social/arabica/internal/web/pages"

	"github.com/rs/zerolog/log"
)

// About page
func (h *Handler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	data, _, _ := h.layoutDataFromRequest(r, "About")

	if err := pages.About(data).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render about page")
	}
}

// Terms of Service page
func (h *Handler) HandleTerms(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.layoutDataFromRequest(r, "Terms of Service")

	if err := pages.Terms(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render terms page")
	}
}

func (h *Handler) HandleATProto(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.layoutDataFromRequest(r, "AT Protocol")

	if err := pages.ATProto(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render AT Protocol page")
	}
}

// Settings page
func (h *Handler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	data, _, isAuthenticated := h.layoutDataFromRequest(r, "Settings")
	if !isAuthenticated {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	var statsVis models.ProfileStatsVisibility
	if h.feedIndex != nil {
		didStr, _ := atproto.GetAuthenticatedDID(r.Context())
		statsVis = h.feedIndex.GetProfileStatsVisibility(r.Context(), didStr)
	} else {
		statsVis = models.DefaultProfileStatsVisibility()
	}

	if err := pages.Settings(data, pages.SettingsProps{
		ProfileStatsVisibility: statsVis,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render settings page")
	}
}

func (h *Handler) HandleSettingsProfileVisibility(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	settings := models.ProfileStatsVisibility{
		BeanAvgRating:    models.Visibility(r.FormValue("bean_avg_rating")),
		RoasterAvgRating: models.Visibility(r.FormValue("roaster_avg_rating")),
	}

	// Validate — fall back to public for unrecognized values
	if !settings.BeanAvgRating.IsValid() {
		settings.BeanAvgRating = models.VisibilityPublic
	}
	if !settings.RoasterAvgRating.IsValid() {
		settings.RoasterAvgRating = models.VisibilityPublic
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
	layoutData, _, _ := h.layoutDataFromRequest(r, "Page Not Found")

	w.WriteHeader(http.StatusNotFound)
	if err := pages.NotFound(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render 404 page")
	}
}
