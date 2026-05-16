package handlers

import (
	"context"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	atp "tangled.org/pdewey.com/atp"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"
)

// Oolong CRUD handlers. Each Create handler decodes the request, builds
// a typed model, calls oolong.XToRecord to produce the PDS record map,
// and writes it via the generic store.PutRecord. Update follows the
// same path with a non-empty rkey. Delete extracts the rkey from the
// URL and calls store.RemoveRecord.

// putOolongRecord is the shared write path for all oolong CRUD Create
// and Update endpoints.
func putOolongRecord(
	ctx context.Context,
	store *atproto.AtprotoStore,
	nsid, rkey string,
	encode func(s *atproto.AtprotoStore) (map[string]any, error),
) (resultRKey string, err error) {
	rec, err := encode(store)
	if err != nil {
		return "", err
	}
	newRKey, _, err := store.PutRecord(ctx, nsid, rkey, rec)
	if err != nil {
		return "", err
	}
	if newRKey == "" {
		newRKey = rkey
	}
	return newRKey, nil
}

func (h *Handler) requireOolongStore(w http.ResponseWriter, r *http.Request) (*atproto.AtprotoStore, bool) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return nil, false
	}
	atpStore, ok := store.(*atproto.AtprotoStore)
	if !ok {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return nil, false
	}
	return atpStore, true
}

// --- Tea -------------------------------------------------------------

func (h *Handler) HandleTeaCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateTeaRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tea := teaFromCreateRequest(&req)
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDTea, "", func(s *atproto.AtprotoStore) (map[string]any, error) {
		var vendorURI string
		if req.VendorRKey != "" {
			vendorURI = atp.BuildATURI(s.DID(), oolong.NSIDVendor, req.VendorRKey)
		}
		return oolong.TeaToRecord(tea, vendorURI)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tea")
		handleStoreError(w, err, "Failed to create tea")
		return
	}
	tea.RKey = rkey
	h.invalidateFeedCache()
	if redirect := r.FormValue("__redirect"); redirect != "" {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, tea, "tea")
}

func (h *Handler) HandleTeaUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.UpdateTeaRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	createReq := oolong.CreateTeaRequest(req)
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tea := teaFromCreateRequest(&createReq)
	tea.RKey = rkey
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDTea, rkey, func(s *atproto.AtprotoStore) (map[string]any, error) {
		var vendorURI string
		if createReq.VendorRKey != "" {
			vendorURI = atp.BuildATURI(s.DID(), oolong.NSIDVendor, createReq.VendorRKey)
		}
		return oolong.TeaToRecord(tea, vendorURI)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update tea")
		handleStoreError(w, err, "Failed to update tea")
		return
	}
	h.invalidateFeedCache()
	if redirect := r.FormValue("__redirect"); redirect != "" {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, tea, "tea")
}

func (h *Handler) HandleTeaDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDTea, "tea")
}

func teaFromCreateRequest(req *oolong.CreateTeaRequest) *oolong.Tea {
	return &oolong.Tea{
		Name:        req.Name,
		Category:    req.Category,
		Origin:      req.Origin,
		HarvestYear: req.HarvestYear,
		Description: req.Description,
		VendorRKey:  req.VendorRKey,
		Rating:      req.Rating,
		Closed:      req.Closed,
		SourceRef:   req.SourceRef,
		CreatedAt:   time.Now(),
	}
}

// --- Vendor ----------------------------------------------------------

func (h *Handler) HandleOolongVendorCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateVendorRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v := &oolong.Vendor{
		Name:        req.Name,
		Location:    req.Location,
		Website:     req.Website,
		Description: req.Description,
		SourceRef:   req.SourceRef,
		CreatedAt:   time.Now(),
	}
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDVendor, "", func(*atproto.AtprotoStore) (map[string]any, error) {
		return oolong.VendorToRecord(v)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create vendor")
		handleStoreError(w, err, "Failed to create vendor")
		return
	}
	v.RKey = rkey
	h.invalidateFeedCache()
	writeJSON(w, v, "vendor")
}

func (h *Handler) HandleOolongVendorUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.UpdateVendorRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	createReq := oolong.CreateVendorRequest{
		Name: req.Name, Location: req.Location, Website: req.Website,
		Description: req.Description, SourceRef: req.SourceRef,
	}
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v := &oolong.Vendor{
		RKey: rkey,
		Name: createReq.Name, Location: createReq.Location, Website: createReq.Website,
		Description: createReq.Description, SourceRef: createReq.SourceRef,
		CreatedAt: time.Now(),
	}
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDVendor, rkey, func(*atproto.AtprotoStore) (map[string]any, error) {
		return oolong.VendorToRecord(v)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update vendor")
		handleStoreError(w, err, "Failed to update vendor")
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, v, "vendor")
}

func (h *Handler) HandleOolongVendorDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDVendor, "vendor")
}

// --- Vessel ----------------------------------------------------------

func (h *Handler) HandleOolongVesselCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateVesselRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v := &oolong.Vessel{
		Name: req.Name, Style: req.Style, CapacityMl: req.CapacityMl,
		Material: req.Material, Description: req.Description,
		SourceRef: req.SourceRef, CreatedAt: time.Now(),
	}
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDVessel, "", func(*atproto.AtprotoStore) (map[string]any, error) {
		return oolong.VesselToRecord(v)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create vessel")
		handleStoreError(w, err, "Failed to create vessel")
		return
	}
	v.RKey = rkey
	h.invalidateFeedCache()
	writeJSON(w, v, "vessel")
}

func (h *Handler) HandleOolongVesselUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.UpdateVesselRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	createReq := oolong.CreateVesselRequest(req)
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	v := &oolong.Vessel{
		RKey: rkey,
		Name: createReq.Name, Style: createReq.Style, CapacityMl: createReq.CapacityMl,
		Material: createReq.Material, Description: createReq.Description,
		SourceRef: createReq.SourceRef, CreatedAt: time.Now(),
	}
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDVessel, rkey, func(*atproto.AtprotoStore) (map[string]any, error) {
		return oolong.VesselToRecord(v)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update vessel")
		handleStoreError(w, err, "Failed to update vessel")
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, v, "vessel")
}

func (h *Handler) HandleOolongVesselDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDVessel, "vessel")
}

// --- Infuser ---------------------------------------------------------

func (h *Handler) HandleOolongInfuserCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateInfuserRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	i := &oolong.Infuser{
		Name: req.Name, Style: req.Style,
		Material: req.Material, Description: req.Description,
		SourceRef: req.SourceRef, CreatedAt: time.Now(),
	}
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDInfuser, "", func(*atproto.AtprotoStore) (map[string]any, error) {
		return oolong.InfuserToRecord(i)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create infuser")
		handleStoreError(w, err, "Failed to create infuser")
		return
	}
	i.RKey = rkey
	h.invalidateFeedCache()
	writeJSON(w, i, "infuser")
}

func (h *Handler) HandleOolongInfuserUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.UpdateInfuserRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	createReq := oolong.CreateInfuserRequest(req)
	if err := createReq.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	i := &oolong.Infuser{
		RKey: rkey,
		Name: createReq.Name, Style: createReq.Style,
		Material: createReq.Material, Description: createReq.Description,
		SourceRef: createReq.SourceRef, CreatedAt: time.Now(),
	}
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDInfuser, rkey, func(*atproto.AtprotoStore) (map[string]any, error) {
		return oolong.InfuserToRecord(i)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update infuser")
		handleStoreError(w, err, "Failed to update infuser")
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, i, "infuser")
}

func (h *Handler) HandleOolongInfuserDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDInfuser, "infuser")
}

// --- Brew (steep) ----------------------------------------------------

func (h *Handler) HandleOolongBrewCreate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req oolong.CreateBrewRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b := brewFromCreateRequest(&req)
	rkey, err := putOolongRecord(r.Context(), store, oolong.NSIDBrew, "", func(s *atproto.AtprotoStore) (map[string]any, error) {
		teaURI := buildOolongRef(s, req.TeaRKey, oolong.NSIDTea)
		vesselURI := buildOolongRef(s, req.VesselRKey, oolong.NSIDVessel)
		infuserURI := buildOolongRef(s, req.InfuserRKey, oolong.NSIDInfuser)
		return oolong.BrewToRecord(b, teaURI, vesselURI, infuserURI)
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create tea brew")
		handleStoreError(w, err, "Failed to create brew")
		return
	}
	b.RKey = rkey
	h.invalidateFeedCache()
	if redirect := r.FormValue("__redirect"); redirect != "" {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, b, "brew")
}

func (h *Handler) HandleOolongBrewUpdate(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	var req oolong.CreateBrewRequest
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b := brewFromCreateRequest(&req)
	b.RKey = rkey
	if _, err := putOolongRecord(r.Context(), store, oolong.NSIDBrew, rkey, func(s *atproto.AtprotoStore) (map[string]any, error) {
		teaURI := buildOolongRef(s, req.TeaRKey, oolong.NSIDTea)
		vesselURI := buildOolongRef(s, req.VesselRKey, oolong.NSIDVessel)
		infuserURI := buildOolongRef(s, req.InfuserRKey, oolong.NSIDInfuser)
		return oolong.BrewToRecord(b, teaURI, vesselURI, infuserURI)
	}); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msg("Failed to update tea brew")
		handleStoreError(w, err, "Failed to update brew")
		return
	}
	h.invalidateFeedCache()
	if redirect := r.FormValue("__redirect"); redirect != "" {
		w.Header().Set("HX-Redirect", redirect)
		w.WriteHeader(http.StatusOK)
		return
	}
	writeJSON(w, b, "brew")
}

func (h *Handler) HandleOolongBrewDelete(w http.ResponseWriter, r *http.Request) {
	h.handleOolongDelete(w, r, oolong.NSIDBrew, "brew")
}

func brewFromCreateRequest(req *oolong.CreateBrewRequest) *oolong.Brew {
	return &oolong.Brew{
		TeaRKey:        req.TeaRKey,
		Style:          req.Style,
		VesselRKey:     req.VesselRKey,
		InfusionMethod: req.InfusionMethod,
		InfuserRKey:    req.InfuserRKey,
		Temperature:    req.Temperature,
		LeafGrams:      req.LeafGrams,
		WaterAmount:    req.WaterAmount,
		TimeSeconds:    req.TimeSeconds,
		TastingNotes:   req.TastingNotes,
		Rating:         req.Rating,
		CreatedAt:      time.Now(),
	}
}

// --- Cafe + Drink ----------------------------------------------------
//
// Cafe and Drink CRUD handlers are deferred for the v1 oolong launch.
// They remain in tree as commented-out skeletons; re-enable when those
// entities are registered as descriptors and the UI flows ship.

// --- Shared helpers ---------------------------------------------------

// handleOolongDelete is the shared body for every oolong Delete handler.
func (h *Handler) handleOolongDelete(w http.ResponseWriter, r *http.Request, nsid, entityName string) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	if err := store.RemoveRecord(r.Context(), nsid, rkey); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Str("nsid", nsid).Msgf("Failed to delete %s", entityName)
		handleStoreError(w, err, "Failed to delete "+entityName)
		return
	}
	h.invalidateFeedCache()
	w.WriteHeader(http.StatusNoContent)
}

// buildOolongRef returns the AT-URI for a referenced entity owned by the
// authenticated user, or "" if the rkey is empty.
func buildOolongRef(s *atproto.AtprotoStore, rkey, nsid string) string {
	if rkey == "" {
		return ""
	}
	return atp.BuildATURI(s.DID(), nsid, rkey)
}

// decodeOolongForm fills a CreateXRequest struct from form-encoded data
// using its `json:"..."` tags as form field names.
func decodeOolongForm(r *http.Request, target any) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	v := reflect.ValueOf(target).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := strings.Split(field.Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		raw := r.FormValue(tag)
		if raw == "" {
			continue
		}
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		switch fv.Kind() {
		case reflect.String:
			fv.SetString(raw)
		case reflect.Int, reflect.Int32, reflect.Int64:
			if n, err := strconv.Atoi(raw); err == nil {
				fv.SetInt(int64(n))
			}
		case reflect.Float32, reflect.Float64:
			if n, err := strconv.ParseFloat(raw, 64); err == nil {
				fv.SetFloat(n)
			}
		case reflect.Bool:
			fv.SetBool(raw == "true" || raw == "on" || raw == "1")
		case reflect.Ptr:
			if fv.Type().Elem().Kind() == reflect.Int {
				if n, err := strconv.Atoi(raw); err == nil {
					fv.Set(reflect.ValueOf(&n))
				}
			}
		}
	}
	return nil
}
