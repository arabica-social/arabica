package handlers

import (
	"net/http"

	"github.com/rs/zerolog/log"
	"tangled.org/arabica.social/arabica/internal/atproto"
)

// oolongValidator is a pointer-constraint that lets the generic CRUD
// helper call Validate() on a value of type Req without copying.
// Every oolong Create/Update request type satisfies this via its
// pointer-receiver Validate() method.
type oolongValidator[T any] interface {
	*T
	Validate() error
}

// oolongCRUDWrite is the shared body for every oolong Create + Update
// handler that follows the standard shape: require store -> decode +
// validate request -> build model -> put record -> writeJSON. rkey is
// empty for create, non-empty for update.
//
// Build receives the decoded request and is expected to construct the
// typed model (e.g. *oolong.Vendor). Encode converts the model to the
// PDS record map; it receives the store so handlers needing AT-URI refs
// can call store.DID()/buildOolongRef. SetRKey writes the PDS-assigned
// rkey back onto the model so the JSON response carries it.
//
// allowRedirect mirrors the HX-Redirect short-circuit that the Tea/Brew
// handlers had; entities without an inline form path pass false.
func oolongCRUDWrite[Req any, PReq oolongValidator[Req], Model any](
	h *Handler,
	w http.ResponseWriter,
	r *http.Request,
	nsid, jsonKey, rkey string,
	build func(req *Req) *Model,
	setRKey func(*Model, string),
	encode func(s *atproto.AtprotoStore, req *Req, m *Model) (map[string]any, error),
	allowRedirect bool,
) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	var req Req
	if err := decodeRequest(r, &req, func() error { return decodeOolongForm(r, &req) }); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := PReq(&req).Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	model := build(&req)
	newRKey, err := putOolongRecord(r.Context(), store, nsid, rkey, func(s *atproto.AtprotoStore) (map[string]any, error) {
		return encode(s, &req, model)
	})
	if err != nil {
		log.Error().Err(err).Str("rkey", rkey).Str("nsid", nsid).Msgf("Failed to write %s", jsonKey)
		handleStoreError(w, err, "Failed to save "+jsonKey)
		return
	}
	setRKey(model, newRKey)
	h.invalidateFeedCache()
	if allowRedirect {
		if redirect := r.FormValue("__redirect"); redirect != "" {
			w.Header().Set("HX-Redirect", redirect)
			w.WriteHeader(http.StatusOK)
			return
		}
	}
	writeJSON(w, model, jsonKey)
}

// updateRKey is a small helper for handlers that need to extract +
// validate the rkey from the URL path before dispatching to
// oolongCRUDWrite. Returns "" when validation already wrote an error
// response (caller should return immediately).
func updateRKey(w http.ResponseWriter, r *http.Request) string {
	return validateRKey(w, r.PathValue("id"))
}
