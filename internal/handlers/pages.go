package handlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	teapages "tangled.org/arabica.social/arabica/internal/oolong/web/pages"
	"tangled.org/arabica.social/arabica/internal/web/pages"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// About page — dispatches to the oolong variant when the running app
// is the tea sister. Both pages share the same layout chrome; only the
// per-app copy and entity-noun references differ.
func (h *Handler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	data, _, _ := h.layoutDataFromRequest(r, "About")

	var err error
	if h.app != nil && h.app.Name == "oolong" {
		err = teapages.About(data).Render(r.Context(), w)
	} else {
		err = pages.About(data).Render(r.Context(), w)
	}
	if err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render about page")
	}
}

// Terms of Service page — dispatches per app for the same reason
// HandleAbout does: brand strings and entity-noun references differ.
func (h *Handler) HandleTerms(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.layoutDataFromRequest(r, "Terms of Service")

	var err error
	if h.app != nil && h.app.Name == "oolong" {
		err = teapages.Terms(layoutData).Render(r.Context(), w)
	} else {
		err = pages.Terms(layoutData).Render(r.Context(), w)
	}
	if err != nil {
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

	didStr, _ := atpmiddleware.GetDID(r.Context())
	sessionID, _ := atpmiddleware.GetSessionID(r.Context())

	var statsVis arabica.ProfileStatsVisibility
	if h.feedIndex != nil {
		statsVis = h.feedIndex.GetProfileStatsVisibility(r.Context(), didStr)
	} else {
		statsVis = arabica.DefaultProfileStatsVisibility()
	}

	bskyForm := h.loadBlueskyProfileForm(r.Context(), didStr, sessionID)

	if err := pages.Settings(data, pages.SettingsProps{
		ProfileStatsVisibility: statsVis,
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

	settings := arabica.ProfileStatsVisibility{
		BeanAvgRating:    arabica.Visibility(r.FormValue("bean_avg_rating")),
		RoasterAvgRating: arabica.Visibility(r.FormValue("roaster_avg_rating")),
	}

	// Validate — fall back to public for unrecognized values
	if !settings.BeanAvgRating.IsValid() {
		settings.BeanAvgRating = arabica.VisibilityPublic
	}
	if !settings.RoasterAvgRating.IsValid() {
		settings.RoasterAvgRating = arabica.VisibilityPublic
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
