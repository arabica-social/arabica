package teahandlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/records"
)

// oolongCRUDWrite is the shared body for every oolong Create + Update
// handler that follows the standard shape: require store -> decode +
// validate request -> build model -> put record -> handlers.WriteJSON. rkey is
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
func oolongCRUDWrite[Req any, PReq handlers.RequestValidator[Req], Model any](
	h *Handlers,
	w http.ResponseWriter,
	r *http.Request,
	nsid, jsonKey, rkey string,
	build func(req *Req) *Model,
	setRKey func(*Model, string),
	encode func(s records.Store, req *Req, m *Model) (map[string]any, error),
	allowRedirect bool,
) {
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	handlers.RecordCRUDWrite[Req, PReq, Model](
		w, r, store, nsid, jsonKey, rkey,
		func(r *http.Request, req *Req) error { return decodeOolongForm(r, req) },
		build, setRKey, encode, h.InvalidateFeedCache, allowRedirect,
	)
}

// updateRKey is a small helper for handlers that need to extract +
// validate the rkey from the URL path before dispatching to
// oolongCRUDWrite. Returns "" when validation already wrote an error
// response (caller should return immediately).
func updateRKey(w http.ResponseWriter, r *http.Request) string {
	return handlers.ValidateRKey(w, r.PathValue("id"))
}
