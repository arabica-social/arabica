package handlers

import (
	"net/http"

	"github.com/rs/zerolog/log"

	"tangled.org/arabica.social/arabica/internal/entities/oolong"
	tea "tangled.org/arabica.social/arabica/internal/oolong/web/components"
)

// Modal-partial handlers for the 7 oolong entities. New variants render
// an empty modal; Edit variants fetch the existing record via the
// generic AtprotoStore.FetchRecord + oolong.RecordTo* decoder, then
// render the modal pre-filled.
//
// The dialog modal templ files in internal/oolong/web/components/ are
// driven by ModalShell, which posts back to /api/{urlpath} for create
// and /api/{urlpath}/{id} for update. The handlers here only render the
// dialog HTML — the actual write goes through oolong_crud.go.

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

// --- Brewer ----------------------------------------------------------

func (h *Handler) HandleOolongBrewerModalNew(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireOolongStore(w, r); !ok {
		return
	}
	if err := tea.BrewerDialogModal(nil).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render tea brewer modal (new)")
	}
}

func (h *Handler) HandleOolongBrewerModalEdit(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), oolong.NSIDBrewer, rkey)
	if err != nil {
		http.Error(w, "Brewer not found", http.StatusNotFound)
		return
	}
	b, err := oolong.RecordToBrewer(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode brewer", http.StatusInternalServerError)
		return
	}
	b.RKey = rkey
	if err := tea.BrewerDialogModal(b).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render tea brewer modal (edit)")
	}
}

// --- Recipe ----------------------------------------------------------

func (h *Handler) HandleOolongRecipeModalNew(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireOolongStore(w, r); !ok {
		return
	}
	if err := tea.RecipeDialogModal(nil).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render tea recipe modal (new)")
	}
}

func (h *Handler) HandleOolongRecipeModalEdit(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), oolong.NSIDRecipe, rkey)
	if err != nil {
		http.Error(w, "Recipe not found", http.StatusNotFound)
		return
	}
	rcp, err := oolong.RecordToRecipe(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode recipe", http.StatusInternalServerError)
		return
	}
	rcp.RKey = rkey
	if err := tea.RecipeDialogModal(rcp).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render tea recipe modal (edit)")
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

// --- Cafe ------------------------------------------------------------

func (h *Handler) HandleOolongCafeModalNew(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireOolongStore(w, r); !ok {
		return
	}
	if err := tea.CafeDialogModal(nil).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render tea cafe modal (new)")
	}
}

func (h *Handler) HandleOolongCafeModalEdit(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), oolong.NSIDCafe, rkey)
	if err != nil {
		http.Error(w, "Cafe not found", http.StatusNotFound)
		return
	}
	c, err := oolong.RecordToCafe(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode cafe", http.StatusInternalServerError)
		return
	}
	c.RKey = rkey
	if err := tea.CafeDialogModal(c).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render tea cafe modal (edit)")
	}
}

// --- Drink -----------------------------------------------------------

func (h *Handler) HandleOolongDrinkModalNew(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.requireOolongStore(w, r); !ok {
		return
	}
	if err := tea.DrinkDialogModal(nil).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render tea drink modal (new)")
	}
}

func (h *Handler) HandleOolongDrinkModalEdit(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), oolong.NSIDDrink, rkey)
	if err != nil {
		http.Error(w, "Drink not found", http.StatusNotFound)
		return
	}
	d, err := oolong.RecordToDrink(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode drink", http.StatusInternalServerError)
		return
	}
	d.RKey = rkey
	if err := tea.DrinkDialogModal(d).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render tea drink modal (edit)")
	}
}
