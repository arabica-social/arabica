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
// returns every record across the 7 oolong entity types the
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
		teas    []*oolong.Tea
		vendors []*oolong.Vendor
		brewers []*oolong.Brewer
		recipes []*oolong.Recipe
		brews   []*oolong.Brew
		cafes   []*oolong.Cafe
		drinks  []*oolong.Drink
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
		brewers = listOolong(gctx, store, oolong.NSIDBrewer, oolong.RecordToBrewer)
		return nil
	})
	g.Go(func() error {
		recipes = listOolong(gctx, store, oolong.NSIDRecipe, oolong.RecordToRecipe)
		return nil
	})
	g.Go(func() error {
		brews = listOolong(gctx, store, oolong.NSIDBrew, oolong.RecordToBrew)
		return nil
	})
	g.Go(func() error {
		cafes = listOolong(gctx, store, oolong.NSIDCafe, oolong.RecordToCafe)
		return nil
	})
	g.Go(func() error {
		drinks = listOolong(gctx, store, oolong.NSIDDrink, oolong.RecordToDrink)
		return nil
	})
	_ = g.Wait()

	response := map[string]any{
		"did":     userDID,
		"teas":    teas,
		"vendors": vendors,
		"brewers": brewers,
		"recipes": recipes,
		"brews":   brews,
		"cafes":   cafes,
		"drinks":  drinks,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode oolong API response")
	}
}
