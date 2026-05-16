package handlers

import (
	"context"

	atp "tangled.org/pdewey.com/atp"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
)

// standardViewTriple builds the fromWitness/fromPDS/fromStore lambdas
// for an entity view config from a record decoder. The three paths
// (witness cache, PDS public client, authenticated own-store) all
// share the same decode-and-set-rkey shape; per-entity ref resolution
// belongs in entityViewConfig.resolveRefs, not in these lambdas.
func standardViewTriple[M any](
	nsid string,
	decode func(map[string]any, string) (*M, error),
	setRKey func(*M, string),
) (
	fromWitness func(ctx context.Context, m map[string]any, uri, rkey, ownerDID string) (any, error),
	fromPDS func(ctx context.Context, e *atp.Record, rkey, ownerDID string) (any, error),
	fromStore func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, map[string]any, string, string, error),
) {
	fromWitness = func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
		model, err := decode(m, uri)
		if err != nil {
			return nil, err
		}
		setRKey(model, rkey)
		return model, nil
	}
	fromPDS = func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
		model, err := decode(e.Value, e.URI)
		if err != nil {
			return nil, err
		}
		setRKey(model, rkey)
		return model, nil
	}
	fromStore = func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, map[string]any, string, string, error) {
		raw, uri, cid, err := s.FetchRecord(ctx, nsid, rkey)
		if err != nil {
			return nil, nil, "", "", err
		}
		model, err := decode(raw, uri)
		if err != nil {
			return nil, nil, "", "", err
		}
		setRKey(model, rkey)
		return model, raw, uri, cid, nil
	}
	return
}

// witnessLookup returns a lookup closure suitable for resolveRefs that
// reads foreign records from the witness cache. Returns false if the
// witness cache is unavailable or the URI is not indexed.
func (h *Handler) witnessLookup(ctx context.Context) func(refURI string) (map[string]any, bool) {
	return func(refURI string) (map[string]any, bool) {
		if h.witnessCache == nil || refURI == "" {
			return nil, false
		}
		wr, _ := h.witnessCache.GetWitnessRecord(ctx, refURI)
		if wr == nil {
			return nil, false
		}
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			return nil, false
		}
		return m, true
	}
}

// resolveBrewRefsViaLookup walks a brew's foreign references
// (bean+roaster, grinder, brewer, recipe+brewer) and populates the
// typed fields on the brew via the supplied lookup. The caller binds
// lookup to a specific source (witness cache, public PDS client, etc.)
// so this function is source-agnostic.
//
// Idempotent: each branch checks whether the target field is already
// populated and skips lookup work when so. This matters because the
// own-store path resolves refs at the store layer before this runs,
// and we don't want to redo or clobber that work.
func resolveBrewRefsViaLookup(brew *arabica.Brew, raw map[string]any, lookup func(refURI string) (map[string]any, bool)) {
	if brew == nil || raw == nil {
		return
	}

	// Bean + nested roaster.
	if brew.Bean == nil {
		if beanRef, _ := raw["beanRef"].(string); beanRef != "" {
			if m, ok := lookup(beanRef); ok {
				if bean, err := arabica.RecordToBean(m, beanRef); err == nil {
					if rk := atp.RKeyFromURI(beanRef); rk != "" {
						bean.RKey = rk
					}
					if roasterRef, _ := m["roasterRef"].(string); roasterRef != "" {
						if rk := atp.RKeyFromURI(roasterRef); rk != "" {
							bean.RoasterRKey = rk
						}
						if rm, ok := lookup(roasterRef); ok {
							if roaster, err := arabica.RecordToRoaster(rm, roasterRef); err == nil {
								roaster.RKey = bean.RoasterRKey
								bean.Roaster = roaster
							}
						}
					}
					brew.Bean = bean
				}
			}
		}
	}

	// Grinder.
	if brew.GrinderObj == nil {
		if grinderRef, _ := raw["grinderRef"].(string); grinderRef != "" {
			if m, ok := lookup(grinderRef); ok {
				if grinder, err := arabica.RecordToGrinder(m, grinderRef); err == nil {
					if rk := atp.RKeyFromURI(grinderRef); rk != "" {
						grinder.RKey = rk
					}
					brew.GrinderObj = grinder
				}
			}
		}
	}

	// Brewer.
	if brew.BrewerObj == nil {
		if brewerRef, _ := raw["brewerRef"].(string); brewerRef != "" {
			if m, ok := lookup(brewerRef); ok {
				if brewer, err := arabica.RecordToBrewer(m, brewerRef); err == nil {
					if rk := atp.RKeyFromURI(brewerRef); rk != "" {
						brewer.RKey = rk
					}
					brew.BrewerObj = brewer
				}
			}
		}
	}

	// Recipe + nested brewer.
	if brew.RecipeObj == nil {
		if recipeRef, _ := raw["recipeRef"].(string); recipeRef != "" {
			if m, ok := lookup(recipeRef); ok {
				if recipe, err := arabica.RecordToRecipe(m, recipeRef); err == nil {
					if rk := atp.RKeyFromURI(recipeRef); rk != "" {
						recipe.RKey = rk
					}
					if brewerRef, _ := m["brewerRef"].(string); brewerRef != "" {
						if rk := atp.RKeyFromURI(brewerRef); rk != "" {
							recipe.BrewerRKey = rk
						}
						if bm, ok := lookup(brewerRef); ok {
							if brewer, err := arabica.RecordToBrewer(bm, brewerRef); err == nil {
								brewer.RKey = recipe.BrewerRKey
								recipe.BrewerObj = brewer
							}
						}
					}
					brew.RecipeObj = recipe
				}
			}
		}
	}
}

// publicLookup returns a lookup closure suitable for resolveRefs that
// reads foreign records through the unauthenticated PDS public client.
// Use this on the PDS fallback path where the witness cache may be
// stale or missing.
func publicLookup(ctx context.Context) func(refURI string) (map[string]any, bool) {
	pub := atproto.NewPublicClient()
	return func(refURI string) (map[string]any, bool) {
		if refURI == "" {
			return nil, false
		}
		parsed, err := atp.ParseATURI(refURI)
		if err != nil {
			return nil, false
		}
		rec, err := pub.GetPublicRecord(ctx, parsed.DID, parsed.Collection, parsed.RKey)
		if err != nil || rec == nil {
			return nil, false
		}
		return rec.Value, true
	}
}

