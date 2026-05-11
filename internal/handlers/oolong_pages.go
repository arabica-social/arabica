package handlers

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"
	teapages "tangled.org/arabica.social/arabica/internal/oolong/web/pages"
)

// HandleMyTea renders /my-tea — the oolong equivalent of arabica's
// /my-coffee. Fetches all 7 entity types the authenticated user owns
// and lists them in collapsible sections. Each section surfaces a
// "+ Add" button that opens the matching dialog modal.
//
// Lists come from store.FetchAllRecords (witness-cache-first) + the
// matching oolong.RecordTo* decoder, not from typed Get* wrappers
// (which don't exist for oolong yet). Reference fields (Vendor on
// Tea, etc.) are not joined here — the row label uses the entity's
// own Name or a sensible fallback.
func (h *Handler) HandleMyTea(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}

	props := teapages.MyTeaProps{
		Teas:    listOolong(r.Context(), store, oolong.NSIDTea, oolong.RecordToTea),
		Vendors: listOolong(r.Context(), store, oolong.NSIDVendor, oolong.RecordToVendor),
		Brewers: listOolong(r.Context(), store, oolong.NSIDBrewer, oolong.RecordToBrewer),
		Recipes: listOolong(r.Context(), store, oolong.NSIDRecipe, oolong.RecordToRecipe),
		Brews:   listOolong(r.Context(), store, oolong.NSIDBrew, oolong.RecordToBrew),
		Cafes:   listOolong(r.Context(), store, oolong.NSIDCafe, oolong.RecordToCafe),
		Drinks:  listOolong(r.Context(), store, oolong.NSIDDrink, oolong.RecordToDrink),
	}

	layoutData, _, _ := h.layoutDataFromRequest(r, "My Tea")
	if err := teapages.MyTea(layoutData, props).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render my-tea page")
	}
}

// listOolong fetches every record of nsid the user owns and decodes
// each one. Records that fail to decode are logged and skipped — the
// page degrades to "this record is missing" rather than 500ing.
func listOolong[T any](
	ctx context.Context,
	store *atproto.AtprotoStore,
	nsid string,
	decode func(map[string]any, string) (*T, error),
) []*T {
	raw, err := store.FetchAllRecords(ctx, nsid)
	if err != nil {
		log.Warn().Err(err).Str("nsid", nsid).Msg("FetchAllRecords failed; rendering empty list")
		return nil
	}
	out := make([]*T, 0, len(raw))
	for _, r := range raw {
		t, err := decode(r.Record, r.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", r.URI).Msg("decode failed; skipping record")
			continue
		}
		out = append(out, t)
	}
	return out
}
