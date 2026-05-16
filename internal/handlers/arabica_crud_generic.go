package handlers

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"

	"tangled.org/arabica.social/arabica/internal/database"
)

// arabicaValidator is a pointer-constraint so the generic factories
// below can call Validate() on a value of type Req without copying.
type arabicaValidator[T any] interface {
	*T
	Validate() error
}

// arabicaCRUDCreate is the shared body for arabica Create handlers
// that follow the standard: require store -> decode (JSON or form) ->
// validate -> call typed store.CreateX -> invalidate cache -> writeJSON.
//
// decodeForm fills the request from r.Form for non-JSON requests
// (arabica handlers use bespoke field-name mappings rather than
// reflection-based decoding). create calls the typed store method.
func arabicaCRUDCreate[Req any, PReq arabicaValidator[Req], Model any](
	h *Handler,
	w http.ResponseWriter,
	r *http.Request,
	name, jsonKey string,
	decodeForm func(r *http.Request) Req,
	create func(ctx context.Context, store database.Store, req *Req) (*Model, error),
) {
	store, authenticated := h.getAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	var req Req
	if err := decodeRequest(r, &req, func() error {
		req = decodeForm(r)
		return nil
	}); err != nil {
		log.Warn().Err(err).Msgf("Failed to decode %s create request", name)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := PReq(&req).Validate(); err != nil {
		log.Warn().Err(err).Msgf("%s create validation failed", name)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	model, err := create(r.Context(), store, &req)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to create %s", name)
		handleStoreError(w, err, "Failed to create "+name)
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, model, jsonKey)
}

// arabicaCRUDUpdate is the shared body for arabica Update handlers.
// After applying the update, the entity is re-fetched so the JSON
// response carries the freshly-persisted record (matching the
// pre-refactor behavior of the hand-cloned handlers).
func arabicaCRUDUpdate[Req any, PReq arabicaValidator[Req], Model any](
	h *Handler,
	w http.ResponseWriter,
	r *http.Request,
	name, jsonKey string,
	decodeForm func(r *http.Request) Req,
	update func(ctx context.Context, store database.Store, rkey string, req *Req) error,
	refetch func(ctx context.Context, store database.Store, rkey string) (*Model, error),
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
	var req Req
	if err := decodeRequest(r, &req, func() error {
		req = decodeForm(r)
		return nil
	}); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msgf("Failed to decode %s update request", name)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := PReq(&req).Validate(); err != nil {
		log.Warn().Err(err).Str("rkey", rkey).Msgf("%s update validation failed", name)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := update(r.Context(), store, rkey, &req); err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msgf("Failed to update %s", name)
		handleStoreError(w, err, "Failed to update "+name)
		return
	}
	model, err := refetch(r.Context(), store, rkey)
	if err != nil {
		log.Error().Err(err).Str("rkey", rkey).Msgf("Failed to get %s after update", name)
		http.Error(w, "Failed to fetch updated "+name, http.StatusInternalServerError)
		return
	}
	h.invalidateFeedCache()
	writeJSON(w, model, jsonKey)
}
