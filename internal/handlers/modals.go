package handlers

import (
	"net/http"

	"arabica/internal/models"
	"arabica/internal/web/components"

	"github.com/rs/zerolog/log"
)

// Modal dialog handlers for entity management

// HandleBeanModalNew renders a new bean modal dialog
func (h *Handler) HandleBeanModalNew(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch roasters for the select dropdown
	roasters, err := store.ListRoasters(r.Context())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch roasters for bean modal")
		roasters = []*models.Roaster{} // Empty list on error
	}

	// Convert to slice for template
	roastersSlice := make([]models.Roaster, len(roasters))
	for i, r := range roasters {
		roastersSlice[i] = *r
	}

	if err := components.BeanDialogModal(nil, roastersSlice).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render bean modal")
	}
}

// HandleBeanModalEdit renders an edit bean modal dialog
func (h *Handler) HandleBeanModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch the bean
	bean, err := store.GetBeanByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Bean not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get bean for modal")
		return
	}

	// Fetch roasters for the select dropdown
	roasters, err := store.ListRoasters(r.Context())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch roasters for bean modal")
		roasters = []*models.Roaster{}
	}

	// Convert to slice for template
	roastersSlice := make([]models.Roaster, len(roasters))
	for i, r := range roasters {
		roastersSlice[i] = *r
	}

	if err := components.BeanDialogModal(bean, roastersSlice).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render bean modal")
	}
}

// HandleGrinderModalNew renders a new grinder modal dialog
func (h *Handler) HandleGrinderModalNew(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := components.GrinderDialogModal(nil).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render grinder modal")
	}
}

// HandleGrinderModalEdit renders an edit grinder modal dialog
func (h *Handler) HandleGrinderModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch the grinder
	grinder, err := store.GetGrinderByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Grinder not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get grinder for modal")
		return
	}

	if err := components.GrinderDialogModal(grinder).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render grinder modal")
	}
}

// HandleBrewerModalNew renders a new brewer modal dialog
func (h *Handler) HandleBrewerModalNew(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := components.BrewerDialogModal(nil).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brewer modal")
	}
}

// HandleBrewerModalEdit renders an edit brewer modal dialog
func (h *Handler) HandleBrewerModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch the brewer
	brewer, err := store.GetBrewerByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Brewer not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get brewer for modal")
		return
	}

	if err := components.BrewerDialogModal(brewer).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render brewer modal")
	}
}

// HandleRoasterModalNew renders a new roaster modal dialog
func (h *Handler) HandleRoasterModalNew(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if err := components.RoasterDialogModal(nil).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render roaster modal")
	}
}

// HandleRoasterModalEdit renders an edit roaster modal dialog
func (h *Handler) HandleRoasterModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}

	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Fetch the roaster
	roaster, err := store.GetRoasterByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Roaster not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get roaster for modal")
		return
	}

	if err := components.RoasterDialogModal(roaster).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render roaster modal")
	}
}
