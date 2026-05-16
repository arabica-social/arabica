package handlers

import (
	"net/http"

	"github.com/rs/zerolog/log"

	"tangled.org/arabica.social/arabica/internal/entities/oolong"
	tea "tangled.org/arabica.social/arabica/internal/oolong/web/components"
)

// Modal-partial handlers for the oolong entities. New variants render
// an empty modal; Edit variants fetch the existing record via the
// generic AtprotoStore.FetchRecord + oolong.RecordTo* decoder, then
// render the modal pre-filled.

// Tea uses a full-page form (HandleOolongTeaNew/Edit) instead of a
// modal — see internal/oolong/web/pages/tea_form.templ.

// --- Vendor ----------------------------------------------------------

func (h *Handler) HandleOolongVendorModalNew(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireOolongStore(w, r); !ok {
		return
	}
	if err := tea.VendorDialogModal(nil).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render vendor modal (new)")
	}
}

func (h *Handler) HandleOolongVendorModalEdit(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), oolong.NSIDVendor, rkey)
	if err != nil {
		http.Error(w, "Vendor not found", http.StatusNotFound)
		return
	}
	v, err := oolong.RecordToVendor(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode vendor", http.StatusInternalServerError)
		return
	}
	v.RKey = rkey
	if err := tea.VendorDialogModal(v).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render vendor modal (edit)")
	}
}

// --- Vessel ----------------------------------------------------------

func (h *Handler) HandleOolongVesselModalNew(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireOolongStore(w, r); !ok {
		return
	}
	if err := tea.VesselDialogModal(nil).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render vessel modal (new)")
	}
}

func (h *Handler) HandleOolongVesselModalEdit(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), oolong.NSIDVessel, rkey)
	if err != nil {
		http.Error(w, "Vessel not found", http.StatusNotFound)
		return
	}
	v, err := oolong.RecordToVessel(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode vessel", http.StatusInternalServerError)
		return
	}
	v.RKey = rkey
	if err := tea.VesselDialogModal(v).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render vessel modal (edit)")
	}
}

// --- Infuser ---------------------------------------------------------

func (h *Handler) HandleOolongInfuserModalNew(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireOolongStore(w, r); !ok {
		return
	}
	if err := tea.InfuserDialogModal(nil).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render infuser modal (new)")
	}
}

func (h *Handler) HandleOolongInfuserModalEdit(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), oolong.NSIDInfuser, rkey)
	if err != nil {
		http.Error(w, "Infuser not found", http.StatusNotFound)
		return
	}
	i, err := oolong.RecordToInfuser(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode infuser", http.StatusInternalServerError)
		return
	}
	i.RKey = rkey
	if err := tea.InfuserDialogModal(i).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render infuser modal (edit)")
	}
}

// --- Brew (steep) ----------------------------------------------------

func (h *Handler) HandleOolongBrewModalNew(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireOolongStore(w, r); !ok {
		return
	}
	if err := tea.BrewDialogModal(nil).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render tea brew modal (new)")
	}
}

func (h *Handler) HandleOolongBrewModalEdit(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), oolong.NSIDBrew, rkey)
	if err != nil {
		http.Error(w, "Brew not found", http.StatusNotFound)
		return
	}
	b, err := oolong.RecordToBrew(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode brew", http.StatusInternalServerError)
		return
	}
	b.RKey = rkey
	if err := tea.BrewDialogModal(b).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render tea brew modal (edit)")
	}
}

// Cafe and Drink modal handlers are deferred for v1.
