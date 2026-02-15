package handlers

import (
	"net/http"

	"arabica/internal/atproto"
	"arabica/internal/web/bff"
	"arabica/internal/web/pages"

	"github.com/rs/zerolog/log"
)

// About page
func (h *Handler) HandleAbout(w http.ResponseWriter, r *http.Request) {
	// Check if user is authenticated
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	data := h.buildLayoutData(r, "About", isAuthenticated, didStr, userProfile)

	// Use templ component
	if err := pages.About(data).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render about page")
	}
}

// Terms of Service page
func (h *Handler) HandleTerms(w http.ResponseWriter, r *http.Request) {
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	layoutData := h.buildLayoutData(r, "Terms of Service", isAuthenticated, didStr, userProfile)

	if err := pages.Terms(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render terms page")
	}
}

func (h *Handler) HandleATProto(w http.ResponseWriter, r *http.Request) {
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	layoutData := h.buildLayoutData(r, "AT Protocol", isAuthenticated, didStr, userProfile)

	if err := pages.ATProto(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render AT Protocol page")
	}
}

// HandleNotFound renders the 404 page
func (h *Handler) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	// Check if current user is authenticated (for nav bar state)
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	isAuthenticated := err == nil && didStr != ""

	var userProfile *bff.UserProfile
	if isAuthenticated {
		userProfile = h.getUserProfile(r.Context(), didStr)
	}

	layoutData := h.buildLayoutData(r, "Page Not Found", isAuthenticated, didStr, userProfile)

	w.WriteHeader(http.StatusNotFound)
	if err := pages.NotFound(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render 404 page")
	}
}
