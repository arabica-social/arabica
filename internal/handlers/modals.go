package handlers

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/rs/zerolog/log"

	coffee "tangled.org/arabica.social/arabica/internal/arabica/web/components"
	"tangled.org/arabica.social/arabica/internal/database"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
)

// Modal dialog handlers for entity management.
//
// The simple entities (Grinder, Brewer, Roaster) share a one-step
// fetch-then-render flow, factored into arabicaModalNew/Edit below.
// Bean is bespoke because its modal needs the roaster list for the
// select dropdown.

// arabicaModalNew renders an empty (create-mode) modal after asserting
// the caller is authenticated.
func (h *Handler) arabicaModalNew(w http.ResponseWriter, r *http.Request, name string, render func() templ.Component) {
	if _, authenticated := h.getAtprotoStore(r); !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	if err := render().Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Failed to render %s modal", name)
	}
}

// arabicaModalEdit fetches a record by rkey via fetch and renders the
// pre-filled edit modal.
func arabicaModalEdit[Model any](
	h *Handler,
	w http.ResponseWriter,
	r *http.Request,
	name string,
	fetch func(context.Context, database.Store, string) (*Model, error),
	render func(*Model) templ.Component,
) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	m, err := fetch(r.Context(), store, rkey)
	if err != nil {
		http.Error(w, name+" not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msgf("Failed to get %s for modal", name)
		return
	}
	if err := render(m).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msgf("Failed to render %s modal", name)
	}
}

// --- Bean ------------------------------------------------------------
//
// Bean is bespoke because the modal needs the roaster list for the
// select dropdown.

func (h *Handler) HandleBeanModalNew(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	if err := coffee.BeanDialogModal(nil, beanModalRoasters(r.Context(), store)).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render bean modal")
	}
}

func (h *Handler) HandleBeanModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	bean, err := store.GetBeanByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Bean not found", http.StatusNotFound)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get bean for modal")
		return
	}
	if err := coffee.BeanDialogModal(bean, beanModalRoasters(r.Context(), store)).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render bean modal")
	}
}

func beanModalRoasters(ctx context.Context, store database.Store) []arabica.Roaster {
	roasters, err := store.ListRoasters(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to fetch roasters for bean modal")
		return []arabica.Roaster{}
	}
	out := make([]arabica.Roaster, len(roasters))
	for i, r := range roasters {
		out[i] = *r
	}
	return out
}

// --- Grinder ---------------------------------------------------------

func (h *Handler) HandleGrinderModalNew(w http.ResponseWriter, r *http.Request) {
	h.arabicaModalNew(w, r, "grinder", func() templ.Component { return coffee.GrinderDialogModal(nil) })
}

func (h *Handler) HandleGrinderModalEdit(w http.ResponseWriter, r *http.Request) {
	arabicaModalEdit(h, w, r, "grinder",
		func(ctx context.Context, s database.Store, rkey string) (*arabica.Grinder, error) {
			return s.GetGrinderByRKey(ctx, rkey)
		},
		func(g *arabica.Grinder) templ.Component { return coffee.GrinderDialogModal(g) },
	)
}

// --- Brewer ----------------------------------------------------------

func (h *Handler) HandleBrewerModalNew(w http.ResponseWriter, r *http.Request) {
	h.arabicaModalNew(w, r, "brewer", func() templ.Component { return coffee.BrewerDialogModal(nil) })
}

func (h *Handler) HandleBrewerModalEdit(w http.ResponseWriter, r *http.Request) {
	arabicaModalEdit(h, w, r, "brewer",
		func(ctx context.Context, s database.Store, rkey string) (*arabica.Brewer, error) {
			return s.GetBrewerByRKey(ctx, rkey)
		},
		func(b *arabica.Brewer) templ.Component { return coffee.BrewerDialogModal(b) },
	)
}

// --- Roaster ---------------------------------------------------------

func (h *Handler) HandleRoasterModalNew(w http.ResponseWriter, r *http.Request) {
	h.arabicaModalNew(w, r, "roaster", func() templ.Component { return coffee.RoasterDialogModal(nil) })
}

func (h *Handler) HandleRoasterModalEdit(w http.ResponseWriter, r *http.Request) {
	arabicaModalEdit(h, w, r, "roaster",
		func(ctx context.Context, s database.Store, rkey string) (*arabica.Roaster, error) {
			return s.GetRoasterByRKey(ctx, rkey)
		},
		func(r *arabica.Roaster) templ.Component { return coffee.RoasterDialogModal(r) },
	)
}
