package handlers

import (
	"net/http"

	arabicastore "tangled.org/arabica.social/arabica/internal/arabica/store"
)

func (h *Handler) getArabicaStore(r *http.Request) (arabicastore.Store, bool) {
	store, ok := h.GetRecordStore(r)
	if !ok {
		return nil, false
	}
	arabicaStore, ok := store.(arabicastore.Store)
	if !ok {
		return nil, false
	}
	return arabicaStore, true
}
