package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"tangled.org/arabica.social/arabica/internal/records"
)

// RequireRecordStore returns the authenticated request's app-agnostic record
// store, writing the appropriate HTTP error when auth or store wiring fails.
func (h *Handler) RequireRecordStore(w http.ResponseWriter, r *http.Request) (records.Store, bool) {
	store, authenticated := h.GetAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return nil, false
	}
	recordStore, ok := store.(records.Store)
	if !ok {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return nil, false
	}
	return recordStore, true
}

// ExistingCreatedAt returns the createdAt timestamp of an existing record so
// update handlers can preserve it. It falls back to time.Now when the record
// cannot be fetched or contains an invalid timestamp.
func ExistingCreatedAt(ctx context.Context, store records.Store, nsid, rkey string) time.Time {
	rec, _, _, err := store.FetchRecord(ctx, nsid, rkey)
	if err != nil {
		return time.Now()
	}
	s, ok := rec["createdAt"].(string)
	if !ok {
		return time.Now()
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Now()
	}
	return t
}

// RequestValidator is a pointer constraint for request types with Validate.
type RequestValidator[T any] interface {
	*T
	Validate() error
}

// PutRecord is the shared create/update primitive for handlers that encode a
// typed model into an app record and write it through a generic record store.
func PutRecord(
	ctx context.Context,
	store records.Store,
	nsid, rkey string,
	encode func(records.Store) (map[string]any, error),
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

// RecordCRUDWrite is the common body for standard entity Create + Update
// handlers: decode, validate, build model, encode to AT Protocol record,
// write, invalidate, and respond with JSON or an optional HX redirect.
func RecordCRUDWrite[Req any, PReq RequestValidator[Req], Model any](
	w http.ResponseWriter,
	r *http.Request,
	store records.Store,
	nsid, jsonKey, rkey string,
	decodeForm func(*http.Request, *Req) error,
	build func(req *Req) *Model,
	setRKey func(*Model, string),
	encode func(s records.Store, req *Req, m *Model) (map[string]any, error),
	invalidate func(),
	allowRedirect bool,
) {
	if store == nil {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	var req Req
	if err := DecodeRequest(r, &req, func() error {
		if decodeForm == nil {
			return fmt.Errorf("form decoding is not configured")
		}
		return decodeForm(r, &req)
	}); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := PReq(&req).Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	model := build(&req)
	newRKey, err := PutRecord(r.Context(), store, nsid, rkey, func(s records.Store) (map[string]any, error) {
		return encode(s, &req, model)
	})
	if err != nil {
		log.Error().Err(err).Str("rkey", rkey).Str("nsid", nsid).Msgf("Failed to write %s", jsonKey)
		HandleStoreError(w, err, "Failed to save "+jsonKey)
		return
	}
	setRKey(model, newRKey)
	if invalidate != nil {
		invalidate()
	}
	if allowRedirect {
		if redirect := r.FormValue("__redirect"); redirect != "" {
			w.Header().Set("HX-Redirect", redirect)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	WriteJSON(w, model, jsonKey)
}
