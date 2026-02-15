package handlers

import (
	"net/http"

	"arabica/internal/web/pages"

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

// HandleNotFound renders the 404 page
func (h *Handler) HandleNotFound(w http.ResponseWriter, r *http.Request) {
	layoutData, _, _ := h.layoutDataFromRequest(r, "Page Not Found")

	w.WriteHeader(http.StatusNotFound)
	if err := pages.NotFound(layoutData).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render 404 page")
	}
}
