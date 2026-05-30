package coffeehandlers

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/rs/zerolog/log"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	arabicastore "tangled.org/arabica.social/arabica/internal/arabica/store"
	coffee "tangled.org/arabica.social/arabica/internal/arabica/web/components"
	"tangled.org/arabica.social/arabica/internal/handlers"
)

// Modal dialog handlers for entity management.
//
// The simple entities (Grinder, Brewer, Roaster) share a one-step
// fetch-then-render flow, factored into arabicaModalNew/Edit below.
// Bean is bespoke because its modal needs the roaster list for the
// select dropdown.

// arabicaModalNew renders an empty (create-mode) modal after asserting
// the caller is authenticated.
func (h *Handlers) arabicaModalNew(w http.ResponseWriter, r *http.Request, name string, render func() templ.Component) {
	if _, authenticated := h.GetArabicaStore(r); !authenticated {
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
	h *Handlers,
	w http.ResponseWriter,
	r *http.Request,
	name string,
	fetch func(context.Context, arabicastore.Store, string) (*Model, error),
	render func(*Model) templ.Component,
) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	store, authenticated := h.GetArabicaStore(r)
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

func (h *Handlers) HandleBeanModalNew(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	if err := coffee.BeanDialogModal(nil, beanModalRoasters(r.Context(), store)).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render modal", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render bean modal")
	}
}

func (h *Handlers) HandleBeanModalEdit(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	store, authenticated := h.GetArabicaStore(r)
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

func beanModalRoasters(ctx context.Context, store arabicastore.Store) []arabica.Roaster {
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

func (h *Handlers) HandleGrinderModalNew(w http.ResponseWriter, r *http.Request) {
	h.arabicaModalNew(w, r, "grinder", func() templ.Component { return coffee.GrinderDialogModal(nil) })
}

func (h *Handlers) HandleGrinderModalEdit(w http.ResponseWriter, r *http.Request) {
	arabicaModalEdit(h, w, r, "grinder",
		func(ctx context.Context, s arabicastore.Store, rkey string) (*arabica.Grinder, error) {
			return getGrinder(ctx, s, rkey)
		},
		func(g *arabica.Grinder) templ.Component { return coffee.GrinderDialogModal(g) },
	)
}

// --- Brewer ----------------------------------------------------------

func (h *Handlers) HandleBrewerModalNew(w http.ResponseWriter, r *http.Request) {
	h.arabicaModalNew(w, r, "brewer", func() templ.Component { return coffee.BrewerDialogModal(nil) })
}

func (h *Handlers) HandleBrewerModalEdit(w http.ResponseWriter, r *http.Request) {
	arabicaModalEdit(h, w, r, "brewer",
		func(ctx context.Context, s arabicastore.Store, rkey string) (*arabica.Brewer, error) {
			return getBrewer(ctx, s, rkey)
		},
		func(b *arabica.Brewer) templ.Component { return coffee.BrewerDialogModal(b) },
	)
}

// --- Roaster ---------------------------------------------------------

func (h *Handlers) HandleRoasterModalNew(w http.ResponseWriter, r *http.Request) {
	h.arabicaModalNew(w, r, "roaster", func() templ.Component { return coffee.RoasterDialogModal(nil) })
}

func (h *Handlers) HandleRoasterModalEdit(w http.ResponseWriter, r *http.Request) {
	arabicaModalEdit(h, w, r, "roaster",
		func(ctx context.Context, s arabicastore.Store, rkey string) (*arabica.Roaster, error) {
			return s.GetRoasterByRKey(ctx, rkey)
		},
		func(r *arabica.Roaster) templ.Component { return coffee.RoasterDialogModal(r) },
	)
}
