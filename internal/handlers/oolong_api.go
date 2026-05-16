package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"tangled.org/arabica.social/arabica/internal/entities/oolong"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

// HandleOolongAPIListAll is the oolong analog of HandleAPIListAll: it
// returns every record across the oolong entity types the
// authenticated user owns, in a JSON shape the client-side
// ArabicaCache understands. The combo-select system reads this cache
// to filter user-owned entries during typeahead.
func (h *Handler) HandleOolongAPIListAll(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	userDID, _ := atpmiddleware.GetDID(ctx)

	var (
		teas     []*oolong.Tea
		vendors  []*oolong.Vendor
		vessels  []*oolong.Vessel
		infusers []*oolong.Infuser
		brews    []*oolong.Brew
	)

	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		teas = listOolong(gctx, store, oolong.NSIDTea, oolong.RecordToTea)
		return nil
	})
	g.Go(func() error {
		vendors = listOolong(gctx, store, oolong.NSIDVendor, oolong.RecordToVendor)
		return nil
	})
	g.Go(func() error {
		vessels = listOolong(gctx, store, oolong.NSIDVessel, oolong.RecordToVessel)
		return nil
	})
	g.Go(func() error {
		infusers = listOolong(gctx, store, oolong.NSIDInfuser, oolong.RecordToInfuser)
		return nil
	})
	g.Go(func() error {
		brews = listOolong(gctx, store, oolong.NSIDBrew, oolong.RecordToBrew)
		return nil
	})
	_ = g.Wait()

	response := map[string]any{
		"did":      userDID,
		"teas":     teas,
		"vendors":  vendors,
		"vessels":  vessels,
		"infusers": infusers,
		"brews":    brews,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode oolong API response")
	}
}
