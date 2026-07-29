package handlers

import (
	"context"
	"fmt"
	"net/http"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	coffeeogcard "tangled.org/arabica.social/arabica/internal/arabica/ogcard"
	arabicastore "tangled.org/arabica.social/arabica/internal/arabica/store"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/ogcard"
	"tangled.org/pdewey.com/atp"

	"github.com/rs/zerolog/log"
)

func (h *Handlers) roasterViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, fromStore := handlers.StandardViewTriple(
		arabica.NSIDRoaster, arabica.RecordToRoaster,
		func(r *arabica.Roaster, k string) { r.RKey = k },
	)
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeRoaster),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		DisplayName: func(record any) string { return record.(*arabica.Roaster).Name },
		OGSubtitle:  func(record any) string { return record.(*arabica.Roaster).Name },
		CountLookup: func(ctx context.Context, ownerDID, subjectURI string) int {
			if h.FeedIndex() == nil || subjectURI == "" {
				return 0
			}
			return h.FeedIndex().BeanCountsByRoasterURI(ctx, ownerDID)[subjectURI]
		},
	}
}

func (h *Handlers) grinderViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, fromStore := handlers.StandardViewTriple(
		arabica.NSIDGrinder, arabica.RecordToGrinder,
		func(g *arabica.Grinder, k string) { g.RKey = k },
	)
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeGrinder),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		DisplayName: func(record any) string { return record.(*arabica.Grinder).Name },
		OGSubtitle:  func(record any) string { return record.(*arabica.Grinder).Name },
		CountLookup: func(ctx context.Context, ownerDID, subjectURI string) int {
			if h.FeedIndex() == nil || subjectURI == "" {
				return 0
			}
			return h.FeedIndex().BrewCountsByGrinderURI(ctx, ownerDID)[subjectURI]
		},
	}
}

func (h *Handlers) brewerViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, fromStore := handlers.StandardViewTriple(
		arabica.NSIDBrewer, arabica.RecordToBrewer,
		func(b *arabica.Brewer, k string) { b.RKey = k },
	)
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeBrewer),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		DisplayName: func(record any) string { return record.(*arabica.Brewer).Name },
		OGSubtitle:  func(record any) string { return record.(*arabica.Brewer).Name },
		CountLookup: func(ctx context.Context, ownerDID, subjectURI string) int {
			if h.FeedIndex() == nil || subjectURI == "" {
				return 0
			}
			return h.FeedIndex().BrewCountsByBrewerURI(ctx, ownerDID)[subjectURI]
		},
	}
}

func (h *Handlers) beanViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, fromStore := handlers.StandardViewTriple(
		arabica.NSIDBean, arabica.RecordToBean,
		func(b *arabica.Bean, k string) { b.RKey = k },
	)
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeBean),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		ResolveRefs: func(_ context.Context, model any, raw map[string]any, lookup func(string) (map[string]any, bool)) {
			arabica.HydrateBeanRefs(model.(*arabica.Bean), raw, lookup)
		},
		DisplayName: func(record any) string { return record.(*arabica.Bean).Name },
		OGSubtitle: func(record any) string {
			bean := record.(*arabica.Bean)
			sub := bean.Name
			if sub == "" {
				sub = bean.Origin
			}
			if bean.Roaster != nil && bean.Roaster.Name != "" {
				sub += " from " + bean.Roaster.Name
			}
			return sub
		},
		CountLookup: func(ctx context.Context, ownerDID, subjectURI string) int {
			if h.FeedIndex() == nil || subjectURI == "" {
				return 0
			}
			return h.FeedIndex().BrewCountsByBeanURI(ctx, ownerDID)[subjectURI]
		},
	}
}

// HandleBeanViewJSON returns bean detail data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleBeanViewJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityViewJSON(w, r, h.beanViewConfig())
}

// HandleBeanBacklinksJSON returns backlinks data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleBeanBacklinksJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderBacklinksViewJSON(w, r, h.beanViewConfig())
}

// HandleRoasterViewJSON returns roaster detail data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleRoasterViewJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityViewJSON(w, r, h.roasterViewConfig())
}

// HandleRoasterBacklinksJSON returns backlinks data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleRoasterBacklinksJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderBacklinksViewJSON(w, r, h.roasterViewConfig())
}

// HandleGrinderViewJSON returns grinder detail data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleGrinderViewJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityViewJSON(w, r, h.grinderViewConfig())
}

// HandleGrinderBacklinksJSON returns backlinks data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleGrinderBacklinksJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderBacklinksViewJSON(w, r, h.grinderViewConfig())
}

// HandleBrewerViewJSON returns brewer detail data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleBrewerViewJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityViewJSON(w, r, h.brewerViewConfig())
}

// HandleBrewerBacklinksJSON returns backlinks data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleBrewerBacklinksJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderBacklinksViewJSON(w, r, h.brewerViewConfig())
}

func (h *Handlers) recipeViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, fromStore := handlers.StandardViewTriple(
		arabica.NSIDRecipe, arabica.RecordToRecipe,
		func(r *arabica.Recipe, k string) { r.RKey = k },
	)
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeRecipe),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		ResolveRefs: func(_ context.Context, model any, raw map[string]any, lookup func(string) (map[string]any, bool)) {
			arabica.HydrateRecipeRefs(model.(*arabica.Recipe), raw, lookup)
		},
		DisplayName: func(record any) string { return record.(*arabica.Recipe).Name },
		OGSubtitle:  func(record any) string { return record.(*arabica.Recipe).Name },
		ViewExtras: func(ctx context.Context, record any) map[string]any {
			recipe := record.(*arabica.Recipe)
			if recipe.SourceRef == "" {
				return nil
			}
			srcURI, err := atp.ParseATURI(recipe.SourceRef)
			if err != nil {
				return nil
			}
			sourceOwner := srcURI.DID
			author := ""
			if h.FeedIndex() != nil {
				if profile, err := h.FeedIndex().GetProfile(ctx, srcURI.DID); err == nil && profile != nil {
					sourceOwner = profile.Handle
					if profile.DisplayName != nil && *profile.DisplayName != "" {
						author = *profile.DisplayName
					} else {
						author = profile.Handle
					}
				}
			}
			return map[string]any{
				"source_recipe_url":    fmt.Sprintf("/recipes/%s/%s", sourceOwner, srcURI.RKey),
				"source_recipe_author": author,
			}
		},
	}
}

// HandleRecipeViewJSON returns recipe detail data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleRecipeViewJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityViewJSON(w, r, h.recipeViewConfig())
}

// HandleRecipeBacklinksJSON returns backlinks data as JSON for the SvelteKit SPA.
func (h *Handlers) HandleRecipeBacklinksJSON(w http.ResponseWriter, r *http.Request) {
	h.RenderBacklinksViewJSON(w, r, h.recipeViewConfig())
}

func (h *Handlers) brewViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, _ := handlers.StandardViewTriple(
		arabica.NSIDBrew, arabica.RecordToBrew,
		func(b *arabica.Brew, k string) { b.RKey = k },
	)
	// Custom fromStore that also extracts the rkeys from ref AT-URIs.
	// We deliberately bypass GetBrewRecordByRKey here: that path resolves
	// refs via the authenticated atpClient as a fallback for session-cache
	// hits that aren't in witness yet, but in practice users select refs
	// from existing entities, so witness lookup via resolveRefs covers it.
	// Worst case: a fresh brew whose refs are also fresh-writes shows
	// unresolved refs until the next firehose catchup.
	fromStore := func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, map[string]any, string, string, error) {
		raw, uri, cid, err := s.FetchRecord(ctx, arabica.NSIDBrew, rkey)
		if err != nil {
			return nil, nil, "", "", err
		}
		brew, err := arabica.RecordToBrew(raw, uri)
		if err != nil {
			return nil, nil, "", "", err
		}
		brew.RKey = rkey
		arabicastore.ExtractBrewRefRKeys(brew, raw)
		return brew, raw, uri, cid, nil
	}
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeBrew),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		ResolveRefs: func(_ context.Context, model any, raw map[string]any, lookup func(string) (map[string]any, bool)) {
			brew := model.(*arabica.Brew)
			arabicastore.ExtractBrewRefRKeys(brew, raw)
			arabica.HydrateBrewRefs(brew, raw, lookup)
		},
		DisplayName: func(any) string { return "Brew Details" },
		OGSubtitle: func(record any) string {
			brew := record.(*arabica.Brew)
			var sub string
			if brew.Bean != nil {
				sub = brew.Bean.Name
				if brew.Bean.Roaster != nil && brew.Bean.Roaster.Name != "" {
					sub += " from " + brew.Bean.Roaster.Name
				}
			}
			return sub
		},
	}
}

// HandleBeanOGImage generates a 1200x630 PNG preview card for a bean.
func (h *Handlers) HandleBeanOGImage(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, "owner parameter required", http.StatusBadRequest)
		return
	}

	ownerDID, err := handlers.ResolveOwnerDID(r.Context(), owner)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var bean *arabica.Bean
	beanURI := atp.BuildATURI(ownerDID, arabica.NSIDBean, rkey)
	if h.WitnessCache() != nil {
		if wr, _ := h.WitnessCache().GetWitnessRecord(r.Context(), beanURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if b, err := arabica.RecordToBean(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("bean_og").Inc()
					bean = b
					bean.RKey = rkey
					if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
						if rwr, _ := h.WitnessCache().GetWitnessRecord(r.Context(), roasterRef); rwr != nil {
							if rm, err := atproto.WitnessRecordToMap(rwr); err == nil {
								if roaster, err := arabica.RecordToRoaster(rm, rwr.URI); err == nil {
									bean.Roaster = roaster
								}
							}
						}
					}
				}
			}
		}
	}
	if bean == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues("bean_og").Inc()
		publicClient := atproto.NewPublicClient()
		record, err := publicClient.GetPublicRecord(r.Context(), ownerDID, arabica.NSIDBean, rkey)
		if err != nil {
			http.Error(w, "Bean not found", http.StatusNotFound)
			return
		}
		bean, err = arabica.RecordToBean(record.Value, record.URI)
		if err != nil {
			http.Error(w, "Failed to load bean", http.StatusInternalServerError)
			return
		}
		if roasterRef, ok := record.Value["roasterRef"].(string); ok && roasterRef != "" {
			roasterRKey := atp.RKeyFromURI(roasterRef)
			if roasterRKey != "" {
				if rr, err := publicClient.GetPublicRecord(r.Context(), ownerDID, arabica.NSIDRoaster, roasterRKey); err == nil {
					if roaster, err := arabica.RecordToRoaster(rr.Value, rr.URI); err == nil {
						bean.Roaster = roaster
					}
				}
			}
		}
	}

	card, err := coffeeogcard.DrawBeanCard(bean)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate bean OG image")
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}
	handlers.WriteOGImage(w, card)
}

// HandleRoasterOGImage generates a 1200x630 PNG preview card for a roaster.
func (h *Handlers) HandleRoasterOGImage(w http.ResponseWriter, r *http.Request) {
	h.HandleSimpleOGImage(w, r, handlers.OGImageConfig{
		NSID: arabica.NSIDRoaster, MetricLabel: "roaster_og",
		Convert: func(m map[string]any, uri, rkey string) (any, error) {
			rec, err := arabica.RecordToRoaster(m, uri)
			if err != nil {
				return nil, err
			}
			rec.RKey = rkey
			return rec, nil
		},
		DrawCard: func(rec any) (*ogcard.Card, error) {
			return coffeeogcard.DrawRoasterCard(rec.(*arabica.Roaster))
		},
	})
}

// HandleGrinderOGImage generates a 1200x630 PNG preview card for a grinder.
func (h *Handlers) HandleGrinderOGImage(w http.ResponseWriter, r *http.Request) {
	h.HandleSimpleOGImage(w, r, handlers.OGImageConfig{
		NSID: arabica.NSIDGrinder, MetricLabel: "grinder_og",
		Convert: func(m map[string]any, uri, rkey string) (any, error) {
			rec, err := arabica.RecordToGrinder(m, uri)
			if err != nil {
				return nil, err
			}
			rec.RKey = rkey
			return rec, nil
		},
		DrawCard: func(rec any) (*ogcard.Card, error) {
			return coffeeogcard.DrawGrinderCard(rec.(*arabica.Grinder))
		},
	})
}

// HandleBrewerOGImage generates a 1200x630 PNG preview card for a brewer.
func (h *Handlers) HandleBrewerOGImage(w http.ResponseWriter, r *http.Request) {
	h.HandleSimpleOGImage(w, r, handlers.OGImageConfig{
		NSID: arabica.NSIDBrewer, MetricLabel: "brewer_og",
		Convert: func(m map[string]any, uri, rkey string) (any, error) {
			rec, err := arabica.RecordToBrewer(m, uri)
			if err != nil {
				return nil, err
			}
			rec.RKey = rkey
			return rec, nil
		},
		DrawCard: func(rec any) (*ogcard.Card, error) {
			return coffeeogcard.DrawBrewerCard(rec.(*arabica.Brewer))
		},
	})
}

// HandleRecipeOGImage generates a 1200x630 PNG preview card for a recipe.
func (h *Handlers) HandleRecipeOGImage(w http.ResponseWriter, r *http.Request) {
	rkey := handlers.ValidateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	owner := r.URL.Query().Get("owner")
	if owner == "" {
		http.Error(w, "owner parameter required", http.StatusBadRequest)
		return
	}

	ownerDID, err := handlers.ResolveOwnerDID(r.Context(), owner)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	var recipe *arabica.Recipe
	recipeURI := atp.BuildATURI(ownerDID, arabica.NSIDRecipe, rkey)
	if h.WitnessCache() != nil {
		if wr, _ := h.WitnessCache().GetWitnessRecord(r.Context(), recipeURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				if rec, err := arabica.RecordToRecipe(m, wr.URI); err == nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues("recipe_og").Inc()
					recipe = rec
					recipe.RKey = rkey
					if brewerRef, ok := m["brewerRef"].(string); ok && brewerRef != "" {
						if bwr, _ := h.WitnessCache().GetWitnessRecord(r.Context(), brewerRef); bwr != nil {
							if bm, err := atproto.WitnessRecordToMap(bwr); err == nil {
								if brewer, err := arabica.RecordToBrewer(bm, bwr.URI); err == nil {
									recipe.BrewerObj = brewer
								}
							}
						}
					}
					recipe.Interpolate()
				}
			}
		}
	}
	if recipe == nil {
		metrics.WitnessCacheMissesTotal.WithLabelValues("recipe_og").Inc()
		publicClient := atproto.NewPublicClient()
		record, err := publicClient.GetPublicRecord(r.Context(), ownerDID, arabica.NSIDRecipe, rkey)
		if err != nil {
			http.Error(w, "Recipe not found", http.StatusNotFound)
			return
		}
		recipe, err = arabica.RecordToRecipe(record.Value, record.URI)
		if err != nil {
			http.Error(w, "Failed to load recipe", http.StatusInternalServerError)
			return
		}
		if brewerRef, ok := record.Value["brewerRef"].(string); ok && brewerRef != "" {
			brewerRKey := atp.RKeyFromURI(brewerRef)
			if brewerRKey != "" {
				if br, err := publicClient.GetPublicRecord(r.Context(), ownerDID, arabica.NSIDBrewer, brewerRKey); err == nil {
					if brewer, err := arabica.RecordToBrewer(br.Value, br.URI); err == nil {
						recipe.BrewerObj = brewer
					}
				}
			}
		}
		recipe.Interpolate()
	}

	card, err := coffeeogcard.DrawRecipeCard(recipe)
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate recipe OG image")
		http.Error(w, "Failed to generate image", http.StatusInternalServerError)
		return
	}
	handlers.WriteOGImage(w, card)
}
