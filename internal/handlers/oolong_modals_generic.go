package handlers

import (
	"net/http"

	"github.com/a-h/templ"
	"github.com/rs/zerolog/log"
)

// oolongModalNew renders an empty (create-mode) dialog modal after
// confirming the caller has an authenticated oolong store. The render
// callback is parameterised on a nil model so each entity's modal
// signature can stay typed (e.g. *oolong.Vendor).
func (h *Handler) oolongModalNew(w http.ResponseWriter, r *http.Request, name string, render func() templ.Component) {
	if _, ok := h.requireOolongStore(w, r); !ok {
		return
	}
	if err := render().Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msgf("Failed to render %s modal (new)", name)
	}
}

// oolongModalEdit fetches an existing record by rkey and renders the
// pre-filled edit modal. decode converts the raw record map into the
// typed model; setRKey writes the rkey back; render produces the modal
// component with the decoded model.
func oolongModalEdit[Model any](
	h *Handler,
	w http.ResponseWriter,
	r *http.Request,
	nsid, name string,
	decode func(rec map[string]any, uri string) (*Model, error),
	setRKey func(*Model, string),
	render func(*Model) templ.Component,
) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), nsid, rkey)
	if err != nil {
		http.Error(w, name+" not found", http.StatusNotFound)
		return
	}
	m, err := decode(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode "+name, http.StatusInternalServerError)
		return
	}
	setRKey(m, rkey)
	if err := render(m).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msgf("Failed to render %s modal (edit)", name)
	}
}
