package handlers

import (
	"encoding/json"
	"net/http"

	"arabica/internal/atproto"
	"arabica/internal/models"
	"arabica/internal/web/components"
	"arabica/internal/web/pages"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// Manage page partial (loaded async via HTMX)
func (h *Handler) HandleManagePartial(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()

	// Fetch all collections in parallel using errgroup for proper error handling
	// and automatic context cancellation on first error
	g, ctx := errgroup.WithContext(ctx)

	var beans []*models.Bean
	var roasters []*models.Roaster
	var grinders []*models.Grinder
	var brewers []*models.Brewer

	g.Go(func() error {
		var err error
		beans, err = store.ListBeans(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		roasters, err = store.ListRoasters(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		grinders, err = store.ListGrinders(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = store.ListBrewers(ctx)
		return err
	})

	if err := g.Wait(); err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to fetch manage page data")
		return
	}

	// Link beans to their roasters
	atproto.LinkBeansToRoasters(beans, roasters)

	// Render manage partial
	if err := components.ManagePartial(components.ManagePartialProps{
		Beans:    beans,
		Roasters: roasters,
		Grinders: grinders,
		Brewers:  brewers,
	}).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render content", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render manage partial")
	}
}

// API endpoint to list all user data (beans, roasters, grinders, brewers, brews)
// Used by client-side cache for faster page loads
func (h *Handler) HandleAPIListAll(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Get user DID for cache validation
	userDID, _ := atproto.GetAuthenticatedDID(r.Context())

	ctx := r.Context()

	// Fetch all collections in parallel using errgroup
	g, ctx := errgroup.WithContext(ctx)

	var beans []*models.Bean
	var roasters []*models.Roaster
	var grinders []*models.Grinder
	var brewers []*models.Brewer
	var brews []*models.Brew

	g.Go(func() error {
		var err error
		beans, err = store.ListBeans(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		roasters, err = store.ListRoasters(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		grinders, err = store.ListGrinders(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brewers, err = store.ListBrewers(ctx)
		return err
	})
	g.Go(func() error {
		var err error
		brews, err = store.ListBrews(ctx, 1) // User ID not used with atproto
		return err
	})

	if err := g.Wait(); err != nil {
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to fetch all data for API")
		return
	}

	// Link beans to roasters
	atproto.LinkBeansToRoasters(beans, roasters)

	response := map[string]interface{}{
		"did":      userDID,
		"beans":    beans,
		"roasters": roasters,
		"grinders": grinders,
		"brewers":  brewers,
		"brews":    brews,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode API response")
	}
}

// API endpoint to create bean
func (h *Handler) HandleBeanCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.CreateBeanRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.CreateBeanRequest{
			Name:        r.FormValue("name"),
			Origin:      r.FormValue("origin"),
			RoastLevel:  r.FormValue("roast_level"),
			Process:     r.FormValue("process"),
			Description: r.FormValue("description"),
			RoasterRKey: r.FormValue("roaster_rkey"),
			Closed:      r.FormValue("closed") == "true",
			SourceRef:   r.FormValue("source_ref"),
		}
		log.Debug().
			Str("name", req.Name).
			Str("closed_value", r.FormValue("closed")).
			Bool("closed_parsed", req.Closed).
			Msg("Parsed bean create form")
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode bean create request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("Bean create validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate optional roaster rkey
	if errMsg := validateOptionalRKey(req.RoasterRKey, "Roaster selection"); errMsg != "" {
		log.Warn().Str("roaster_rkey", req.RoasterRKey).Msg("Bean create: invalid roaster rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	bean, err := store.CreateBean(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create bean", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create bean")
		return
	}

	writeJSON(w, bean, "bean")
}

// API endpoint to create roaster
func (h *Handler) HandleRoasterCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.CreateRoasterRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.CreateRoasterRequest{
			Name:      r.FormValue("name"),
			Location:  r.FormValue("location"),
			Website:   r.FormValue("website"),
			SourceRef: r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode roaster create request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("Roaster create validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	roaster, err := store.CreateRoaster(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create roaster", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create roaster")
		return
	}

	writeJSON(w, roaster, "roaster")
}

// Manage page
func (h *Handler) HandleManage(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	_, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	layoutData, _, _ := h.layoutDataFromRequest(r, "Manage")

	// Create manage props
	manageProps := pages.ManageProps{}

	// Render using templ component
	if err := pages.Manage(layoutData, manageProps).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render manage page")
	}
}

// Bean update/delete handlers
func (h *Handler) HandleBeanUpdate(w http.ResponseWriter, r *http.Request) {
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

	var req models.UpdateBeanRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.UpdateBeanRequest{
			Name:        r.FormValue("name"),
			Origin:      r.FormValue("origin"),
			RoastLevel:  r.FormValue("roast_level"),
			Process:     r.FormValue("process"),
			Description: r.FormValue("description"),
			RoasterRKey: r.FormValue("roaster_rkey"),
			Closed:      r.FormValue("closed") == "true",
			SourceRef:   r.FormValue("source_ref"),
		}
		log.Debug().
			Str("rkey", rkey).
			Str("name", req.Name).
			Str("closed_value", r.FormValue("closed")).
			Bool("closed_parsed", req.Closed).
			Msg("Parsed bean update form")
		return nil
	}); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to decode bean update request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Bean update validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate optional roaster rkey
	if errMsg := validateOptionalRKey(req.RoasterRKey, "Roaster selection"); errMsg != "" {
		log.Warn().Str("rkey", rkey).Str("roaster_rkey", req.RoasterRKey).Msg("Bean update: invalid roaster rkey")
		http.Error(w, errMsg, http.StatusBadRequest)
		return
	}

	if err := store.UpdateBeanByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update bean", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update bean")
		return
	}

	bean, err := store.GetBeanByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated bean", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get bean after update")
		return
	}

	writeJSON(w, bean, "bean")
}

func (h *Handler) HandleBeanDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.deleteEntity(w, r, store.DeleteBeanByRKey, "bean")
}

// Roaster update/delete handlers
func (h *Handler) HandleRoasterUpdate(w http.ResponseWriter, r *http.Request) {
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

	var req models.UpdateRoasterRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.UpdateRoasterRequest{
			Name:      r.FormValue("name"),
			Location:  r.FormValue("location"),
			Website:   r.FormValue("website"),
			SourceRef: r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to decode roaster update request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Roaster update validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateRoasterByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update roaster", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update roaster")
		return
	}

	roaster, err := store.GetRoasterByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated roaster", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get roaster after update")
		return
	}

	writeJSON(w, roaster, "roaster")
}

func (h *Handler) HandleRoasterDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.deleteEntity(w, r, store.DeleteRoasterByRKey, "roaster")
}

// Grinder CRUD handlers
func (h *Handler) HandleGrinderCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.CreateGrinderRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.CreateGrinderRequest{
			Name:        r.FormValue("name"),
			GrinderType: r.FormValue("grinder_type"),
			BurrType:    r.FormValue("burr_type"),
			Notes:       r.FormValue("notes"),
			SourceRef:   r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode grinder create request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("Grinder create validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	grinder, err := store.CreateGrinder(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create grinder", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create grinder")
		return
	}

	writeJSON(w, grinder, "grinder")
}

func (h *Handler) HandleGrinderUpdate(w http.ResponseWriter, r *http.Request) {
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

	var req models.UpdateGrinderRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.UpdateGrinderRequest{
			Name:        r.FormValue("name"),
			GrinderType: r.FormValue("grinder_type"),
			BurrType:    r.FormValue("burr_type"),
			Notes:       r.FormValue("notes"),
			SourceRef:   r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to decode grinder update request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Grinder update validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateGrinderByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update grinder", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update grinder")
		return
	}

	grinder, err := store.GetGrinderByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated grinder", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get grinder after update")
		return
	}

	writeJSON(w, grinder, "grinder")
}

func (h *Handler) HandleGrinderDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.deleteEntity(w, r, store.DeleteGrinderByRKey, "grinder")
}

// Brewer CRUD handlers
func (h *Handler) HandleBrewerCreate(w http.ResponseWriter, r *http.Request) {
	// Require authentication
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	var req models.CreateBrewerRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.CreateBrewerRequest{
			Name:        r.FormValue("name"),
			BrewerType:  r.FormValue("brewer_type"),
			Description: r.FormValue("description"),
			SourceRef:   r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Msg("Failed to decode brewer create request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("name", req.Name).Msg("Brewer create validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	brewer, err := store.CreateBrewer(r.Context(), &req)
	if err != nil {
		http.Error(w, "Failed to create brewer", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to create brewer")
		return
	}

	writeJSON(w, brewer, "brewer")
}

func (h *Handler) HandleBrewerUpdate(w http.ResponseWriter, r *http.Request) {
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

	var req models.UpdateBrewerRequest

	// Decode request (JSON or form)
	if err := decodeRequest(r, &req, func() error {
		req = models.UpdateBrewerRequest{
			Name:        r.FormValue("name"),
			BrewerType:  r.FormValue("brewer_type"),
			Description: r.FormValue("description"),
			SourceRef:   r.FormValue("source_ref"),
		}
		return nil
	}); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Failed to decode brewer update request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msg("Brewer update validation failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := store.UpdateBrewerByRKey(r.Context(), rkey, &req); err != nil {
		http.Error(w, "Failed to update brewer", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update brewer")
		return
	}

	brewer, err := store.GetBrewerByRKey(r.Context(), rkey)
	if err != nil {
		http.Error(w, "Failed to fetch updated brewer", http.StatusInternalServerError)
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to get brewer after update")
		return
	}

	writeJSON(w, brewer, "brewer")
}

func (h *Handler) HandleBrewerDelete(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	h.deleteEntity(w, r, store.DeleteBrewerByRKey, "brewer")
}
