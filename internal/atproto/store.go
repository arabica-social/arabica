package atproto

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tangled.org/arabica.social/arabica/internal/database"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/models"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
)

// AtprotoStore implements the database.Store interface using atproto records.
// Context is passed as a parameter to each method rather than stored in the struct,
// following Go best practices for context propagation.
type AtprotoStore struct {
	client       *Client
	did          syntax.DID
	sessionID    string
	cache        *SessionCache
	witnessCache WitnessCache // optional; enables cache-first reads without PDS calls
}

// NewAtprotoStore creates a new atproto store for a specific user session.
// The cache parameter allows for dependency injection and testability.
func NewAtprotoStore(client *Client, did syntax.DID, sessionID string, cache *SessionCache) database.Store {
	return &AtprotoStore{
		client:    client,
		did:       did,
		sessionID: sessionID,
		cache:     cache,
	}
}

// NewAtprotoStoreWithWitness creates a store that uses the witness cache for
// cache-first reads, falling back to the PDS on cache misses.
func NewAtprotoStoreWithWitness(client *Client, did syntax.DID, sessionID string, cache *SessionCache, witness WitnessCache) database.Store {
	return &AtprotoStore{
		client:       client,
		did:          did,
		sessionID:    sessionID,
		cache:        cache,
		witnessCache: witness,
	}
}

// witnessRecordToMap is a package-internal alias for WitnessRecordToMap.
func witnessRecordToMap(wr *WitnessRecord) (map[string]any, error) {
	return WitnessRecordToMap(wr)
}

// getFromWitness fetches a single record by collection+rkey from the witness cache.
// Returns nil when the cache is not configured, the record is not found,
// or the collection was recently written to (dirty).
func (s *AtprotoStore) getFromWitness(ctx context.Context, collection, rkey string) *WitnessRecord {
	if s.witnessCache == nil {
		return nil
	}
	// Skip witness cache for collections with pending writes
	if userCache := s.cache.Get(s.sessionID); userCache.IsDirty(collection) {
		log.Debug().Str("collection", collection).Msg("witness: skipping dirty collection for single record, falling back to PDS")
		return nil
	}
	uri := BuildATURI(s.did.String(), collection, rkey)
	wr, err := s.witnessCache.GetWitnessRecord(ctx, uri)
	if err != nil {
		log.Debug().Err(err).Str("uri", uri).Msg("witness: GetWitnessRecord error")
		return nil
	}
	return wr
}

// listFromWitness returns all cached records for a collection.
// Returns nil when the cache is not configured or returns nothing.
// Skips the witness cache if the collection was recently written to
// (dirty), since the firehose may not have indexed the new record yet.
func (s *AtprotoStore) listFromWitness(ctx context.Context, collection string) []*WitnessRecord {
	if s.witnessCache == nil {
		return nil
	}
	// Skip witness cache for collections with pending writes
	if userCache := s.cache.Get(s.sessionID); userCache.IsDirty(collection) {
		log.Debug().Str("collection", collection).Msg("witness: skipping dirty collection, falling back to PDS")
		return nil
	}
	records, err := s.witnessCache.ListWitnessRecords(ctx, s.did.String(), collection)
	if err != nil {
		log.Debug().Err(err).Str("collection", collection).Msg("witness: ListWitnessRecords error")
		return nil
	}
	if len(records) == 0 {
		return nil
	}
	return records
}

// writeThroughWitness upserts a record into the witness cache after a
// successful PDS write so subsequent reads see the latest data without
// waiting for the firehose to re-index.
func (s *AtprotoStore) writeThroughWitness(collection, rkey, cid string, record any) {
	if s.witnessCache == nil {
		return
	}
	data, err := json.Marshal(record)
	if err != nil {
		log.Warn().Err(err).Str("collection", collection).Str("rkey", rkey).
			Msg("witness write-through: failed to marshal record")
		return
	}
	if err := s.witnessCache.UpsertWitnessRecord(context.Background(), s.did.String(), collection, rkey, cid, data); err != nil {
		log.Warn().Err(err).Str("collection", collection).Str("rkey", rkey).
			Msg("witness write-through: failed to upsert record")
	}
}

// deleteFromWitness removes a record from the witness cache after a
// successful PDS deletion.
func (s *AtprotoStore) deleteFromWitness(collection, rkey string) {
	if s.witnessCache == nil {
		return
	}
	if err := s.witnessCache.DeleteWitnessRecord(context.Background(), s.did.String(), collection, rkey); err != nil {
		log.Warn().Err(err).Str("collection", collection).Str("rkey", rkey).
			Msg("witness write-through: failed to delete record")
	}
}

// getWitnessRecordByURI fetches a single record by full AT-URI from the witness cache.
// Returns nil when the cache is not configured or the record is not found.
func (s *AtprotoStore) getWitnessRecordByURI(ctx context.Context, uri string) *WitnessRecord {
	if s.witnessCache == nil {
		return nil
	}
	wr, err := s.witnessCache.GetWitnessRecord(ctx, uri)
	if err != nil {
		log.Debug().Err(err).Str("uri", uri).Msg("witness: GetWitnessRecord error")
		return nil
	}
	return wr
}

// resolveBrewRefsFromWitness resolves a brew's references (bean, grinder, brewer, recipe)
// entirely from the witness cache, avoiding any PDS calls. Falls back to PDS-based resolution
// only if a witness lookup fails for any referenced record.
func (s *AtprotoStore) resolveBrewRefsFromWitness(ctx context.Context, brew *models.Brew, record map[string]any) {
	// Resolve bean (and its roaster)
	if beanRef, _ := record["beanRef"].(string); beanRef != "" {
		if beanWR := s.getWitnessRecordByURI(ctx, beanRef); beanWR != nil {
			if beanMap, err := witnessRecordToMap(beanWR); err == nil {
				if bean, err := RecordToBean(beanMap, beanWR.URI); err == nil {
					bean.RKey = beanWR.RKey
					// Resolve roaster ref from witness too
					if roasterRef, ok := beanMap["roasterRef"].(string); ok && roasterRef != "" {
						if c, err := ResolveATURI(roasterRef); err == nil {
							bean.RoasterRKey = c.RKey
						}
						if roasterWR := s.getWitnessRecordByURI(ctx, roasterRef); roasterWR != nil {
							if roasterMap, err := witnessRecordToMap(roasterWR); err == nil {
								if roaster, err := RecordToRoaster(roasterMap, roasterWR.URI); err == nil {
									roaster.RKey = roasterWR.RKey
									bean.Roaster = roaster
								}
							}
						}
					}
					brew.Bean = bean
				}
			}
		}
	}

	// Resolve grinder
	if grinderRef, _ := record["grinderRef"].(string); grinderRef != "" {
		if grinderWR := s.getWitnessRecordByURI(ctx, grinderRef); grinderWR != nil {
			if grinderMap, err := witnessRecordToMap(grinderWR); err == nil {
				if grinder, err := RecordToGrinder(grinderMap, grinderWR.URI); err == nil {
					grinder.RKey = grinderWR.RKey
					brew.GrinderObj = grinder
				}
			}
		}
	}

	// Resolve brewer
	if brewerRef, _ := record["brewerRef"].(string); brewerRef != "" {
		if brewerWR := s.getWitnessRecordByURI(ctx, brewerRef); brewerWR != nil {
			if brewerMap, err := witnessRecordToMap(brewerWR); err == nil {
				if brewer, err := RecordToBrewer(brewerMap, brewerWR.URI); err == nil {
					brewer.RKey = brewerWR.RKey
					brew.BrewerObj = brewer
				}
			}
		}
	}

	// Resolve recipe
	if recipeRef, _ := record["recipeRef"].(string); recipeRef != "" {
		if recipeWR := s.getWitnessRecordByURI(ctx, recipeRef); recipeWR != nil {
			if recipeMap, err := witnessRecordToMap(recipeWR); err == nil {
				if recipe, err := RecordToRecipe(recipeMap, recipeWR.URI); err == nil {
					recipe.RKey = recipeWR.RKey
					// Resolve recipe's brewer ref from witness
					if brewerRef, ok := recipeMap["brewerRef"].(string); ok && brewerRef != "" {
						if c, err := ResolveATURI(brewerRef); err == nil {
							recipe.BrewerRKey = c.RKey
						}
						if brewerWR := s.getWitnessRecordByURI(ctx, brewerRef); brewerWR != nil {
							if brewerMap, err := witnessRecordToMap(brewerWR); err == nil {
								if brewer, err := RecordToBrewer(brewerMap, brewerWR.URI); err == nil {
									brewer.RKey = brewerWR.RKey
									recipe.BrewerObj = brewer
								}
							}
						}
					}
					brew.RecipeObj = recipe
				}
			}
		}
	}
}

// ========== Brew Helpers ==========

// ExtractBrewRefRKeys extracts rkeys from AT-URI references in a brew record's raw values.
func ExtractBrewRefRKeys(brew *models.Brew, record map[string]any) {
	if beanRef, _ := record["beanRef"].(string); beanRef != "" {
		if c, err := ResolveATURI(beanRef); err == nil {
			brew.BeanRKey = c.RKey
		}
	}
	if grinderRef, _ := record["grinderRef"].(string); grinderRef != "" {
		if c, err := ResolveATURI(grinderRef); err == nil {
			brew.GrinderRKey = c.RKey
		}
	}
	if brewerRef, _ := record["brewerRef"].(string); brewerRef != "" {
		if c, err := ResolveATURI(brewerRef); err == nil {
			brew.BrewerRKey = c.RKey
		}
	}
	if recipeRef, _ := record["recipeRef"].(string); recipeRef != "" {
		if c, err := ResolveATURI(recipeRef); err == nil {
			brew.RecipeRKey = c.RKey
		}
	}
}

// brewModelFromRequest converts a CreateBrewRequest into a Brew model with the given creation time.
func brewModelFromRequest(req *models.CreateBrewRequest, createdAt time.Time) *models.Brew {
	brew := &models.Brew{
		BeanRKey:     req.BeanRKey,
		RecipeRKey:   req.RecipeRKey,
		GrinderRKey:  req.GrinderRKey,
		BrewerRKey:   req.BrewerRKey,
		Method:       req.Method,
		Temperature:  req.Temperature,
		WaterAmount:  req.WaterAmount,
		CoffeeAmount: req.CoffeeAmount,
		TimeSeconds:  req.TimeSeconds,
		GrindSize:    req.GrindSize,
		TastingNotes: req.TastingNotes,
		Rating:       req.Rating,
		CreatedAt:    createdAt,
	}
	if len(req.Pours) > 0 {
		brew.Pours = make([]*models.Pour, len(req.Pours))
		for i, pour := range req.Pours {
			brew.Pours[i] = &models.Pour{
				WaterAmount: pour.WaterAmount,
				TimeSeconds: pour.TimeSeconds,
			}
		}
	}
	brew.EspressoParams = req.EspressoParams
	brew.PouroverParams = req.PouroverParams
	return brew
}

// ========== Brew Operations ==========

func (s *AtprotoStore) CreateBrew(ctx context.Context, brew *models.CreateBrewRequest, userID int) (*models.Brew, error) {
	// Build AT-URI references from rkeys
	if brew.BeanRKey == "" {
		return nil, fmt.Errorf("bean_rkey is required")
	}

	beanURI := BuildATURI(s.did.String(), NSIDBean, brew.BeanRKey)

	var grinderURI, brewerURI, recipeURI string
	if brew.GrinderRKey != "" {
		grinderURI = BuildATURI(s.did.String(), NSIDGrinder, brew.GrinderRKey)
	}
	if brew.BrewerRKey != "" {
		brewerURI = BuildATURI(s.did.String(), NSIDBrewer, brew.BrewerRKey)
	}
	if brew.RecipeRKey != "" {
		recipeOwner := s.did.String()
		if brew.RecipeOwnerDID != "" {
			recipeOwner = brew.RecipeOwnerDID
		}
		recipeURI = BuildATURI(recipeOwner, NSIDRecipe, brew.RecipeRKey)
	}

	// Convert to models.Brew for record conversion
	brewModel := brewModelFromRequest(brew, time.Now().UTC())

	// Convert to atproto record
	record, err := BrewToRecord(brewModel, beanURI, grinderURI, brewerURI, recipeURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert brew to record: %w", err)
	}

	// Create record in PDS
	output, err := s.client.CreateRecord(ctx, s.did, s.sessionID, &CreateRecordInput{
		Collection: NSIDBrew,
		Record:     record,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create brew record: %w", err)
	}

	// Parse the returned AT-URI to get the rkey
	atURI, err := syntax.ParseATURI(output.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned AT-URI: %w", err)
	}

	// Store the rkey in the model
	rkey := atURI.RecordKey().String()
	brewModel.RKey = rkey

	s.writeThroughWitness(NSIDBrew, rkey, output.CID, record)
	s.cache.InvalidateBrews(s.sessionID)

	// Fetch and resolve references to populate Bean, Grinder, Brewer
	err = ResolveBrewRefs(ctx, s.client, brewModel, beanURI, grinderURI, brewerURI, s.sessionID)
	if err != nil {
		// Non-fatal: return the brew even if we can't resolve refs
		log.Warn().Err(err).Str("brew_rkey", rkey).Msg("Failed to resolve brew references")
	}

	return brewModel, nil
}

func (s *AtprotoStore) GetBrewByRKey(ctx context.Context, rkey string) (*models.Brew, error) {
	// Try witness cache — resolve the brew AND its references from cache
	if wr := s.getFromWitness(ctx, NSIDBrew, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			brew, err := RecordToBrew(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("brew").Inc()
				brew.RKey = rkey
				ExtractBrewRefRKeys(brew, m)
				s.resolveBrewRefsFromWitness(ctx, brew, m)
				return brew, nil
			}
		}
		log.Warn().Err(err).Str("rkey", rkey).Msg("witness: failed to convert brew, falling back to PDS")
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("brew").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDBrew,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brew record: %w", err)
	}

	// Build the AT-URI for this brew
	atURI := BuildATURI(s.did.String(), NSIDBrew, rkey)

	// Convert to models.Brew
	brew, err := RecordToBrew(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert brew record: %w", err)
	}

	// Set the rkey
	brew.RKey = rkey

	// Extract and resolve references
	ExtractBrewRefRKeys(brew, output.Value)
	beanRef, _ := output.Value["beanRef"].(string)
	grinderRef, _ := output.Value["grinderRef"].(string)
	brewerRef, _ := output.Value["brewerRef"].(string)
	err = ResolveBrewRefs(ctx, s.client, brew, beanRef, grinderRef, brewerRef, s.sessionID)
	if err != nil {
		log.Warn().Err(err).Str("brew_rkey", rkey).Msg("Failed to resolve brew references")
	}
	if recipeRef, _ := output.Value["recipeRef"].(string); recipeRef != "" {
		brew.RecipeObj, err = ResolveRecipeRef(ctx, s.client, recipeRef, s.sessionID)
		if err != nil {
			log.Warn().Err(err).Str("brew_rkey", rkey).Msg("Failed to resolve recipe reference")
		}
	}

	return brew, nil
}

// BrewRecord contains a brew with its AT Protocol metadata
type BrewRecord struct {
	Brew *models.Brew
	URI  string
	CID  string
}

// GetBrewRecordByRKey fetches a brew by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetBrewRecordByRKey(ctx context.Context, rkey string) (*BrewRecord, error) {
	// Try witness cache
	if wr := s.getFromWitness(ctx, NSIDBrew, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			brew, err := RecordToBrew(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("brew").Inc()
				brew.RKey = rkey
				ExtractBrewRefRKeys(brew, m)
				s.resolveBrewRefsFromWitness(ctx, brew, m)
				return &BrewRecord{
					Brew: brew,
					URI:  wr.URI,
					CID:  wr.CID,
				}, nil
			}
		}
		log.Warn().Err(err).Str("rkey", rkey).Msg("witness: failed to convert brew record, falling back to PDS")
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("brew").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDBrew,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brew record: %w", err)
	}

	// Build the AT-URI for this brew
	atURI := BuildATURI(s.did.String(), NSIDBrew, rkey)

	// Convert to models.Brew
	brew, err := RecordToBrew(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert brew record: %w", err)
	}

	// Set the rkey
	brew.RKey = rkey

	// Extract and resolve references
	ExtractBrewRefRKeys(brew, output.Value)
	beanRef, _ := output.Value["beanRef"].(string)
	grinderRef, _ := output.Value["grinderRef"].(string)
	brewerRef, _ := output.Value["brewerRef"].(string)
	err = ResolveBrewRefs(ctx, s.client, brew, beanRef, grinderRef, brewerRef, s.sessionID)
	if err != nil {
		log.Warn().Err(err).Str("brew_rkey", rkey).Msg("Failed to resolve brew references")
	}
	if recipeRef, _ := output.Value["recipeRef"].(string); recipeRef != "" {
		brew.RecipeObj, err = ResolveRecipeRef(ctx, s.client, recipeRef, s.sessionID)
		if err != nil {
			log.Warn().Err(err).Str("brew_rkey", rkey).Msg("Failed to resolve recipe reference")
		}
	}

	return &BrewRecord{
		Brew: brew,
		URI:  output.URI,
		CID:  output.CID,
	}, nil
}

func (s *AtprotoStore) ListBrews(ctx context.Context, userID int) ([]*models.Brew, error) {
	// Check cache first
	userCache := s.cache.Get(s.sessionID)
	if userCache != nil && userCache.Brews() != nil && userCache.IsValid() {
		return userCache.Brews(), nil
	}

	var brews []*models.Brew

	// Try witness cache
	if wRecords := s.listFromWitness(ctx, NSIDBrew); wRecords != nil {
		metrics.WitnessCacheHitsTotal.WithLabelValues("brew").Inc()
		brews = make([]*models.Brew, 0, len(wRecords))
		for _, wr := range wRecords {
			m, err := witnessRecordToMap(wr)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to parse brew")
				continue
			}
			brew, err := RecordToBrew(m, wr.URI)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to convert brew")
				continue
			}
			brew.RKey = wr.RKey
			ExtractBrewRefRKeys(brew, m)
			brews = append(brews, brew)
		}
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("brew").Inc()

		// Use ListAllRecords to handle pagination automatically
		output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, NSIDBrew)
		if err != nil {
			return nil, fmt.Errorf("failed to list brew records: %w", err)
		}

		brews = make([]*models.Brew, 0, len(output.Records))

		for _, rec := range output.Records {
			brew, err := RecordToBrew(rec.Value, rec.URI)
			if err != nil {
				log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert brew record")
				continue
			}

			// Extract rkey from URI
			if components, err := ResolveATURI(rec.URI); err == nil {
				brew.RKey = components.RKey
			}

			// Extract rkeys from AT-URI references
			ExtractBrewRefRKeys(brew, rec.Value)

			brews = append(brews, brew)
		}
	}

	// Resolve references using cached data instead of N+1 queries
	// This fetches beans/grinders/brewers/recipes once (from cache if available)
	// then links them to brews in memory
	beans, _ := s.ListBeans(ctx)
	grinders, _ := s.ListGrinders(ctx)
	brewers, _ := s.ListBrewers(ctx)
	roasters, _ := s.ListRoasters(ctx)
	recipes, _ := s.ListRecipes(ctx)

	// Build lookup maps
	beanMap := make(map[string]*models.Bean)
	for _, b := range beans {
		beanMap[b.RKey] = b
	}
	grinderMap := make(map[string]*models.Grinder)
	for _, g := range grinders {
		grinderMap[g.RKey] = g
	}
	brewerMap := make(map[string]*models.Brewer)
	for _, b := range brewers {
		brewerMap[b.RKey] = b
	}
	roasterMap := make(map[string]*models.Roaster)
	for _, r := range roasters {
		roasterMap[r.RKey] = r
	}
	recipeMap := make(map[string]*models.Recipe)
	for _, r := range recipes {
		recipeMap[r.RKey] = r
	}

	// Link references
	for _, brew := range brews {
		if brew.BeanRKey != "" {
			brew.Bean = beanMap[brew.BeanRKey]
			// Also link roaster to bean
			if brew.Bean != nil && brew.Bean.RoasterRKey != "" {
				brew.Bean.Roaster = roasterMap[brew.Bean.RoasterRKey]
			}
		}
		if brew.GrinderRKey != "" {
			brew.GrinderObj = grinderMap[brew.GrinderRKey]
		}
		if brew.BrewerRKey != "" {
			brew.BrewerObj = brewerMap[brew.BrewerRKey]
		}
		if brew.RecipeRKey != "" {
			brew.RecipeObj = recipeMap[brew.RecipeRKey]
		}
	}

	// Update cache and clear dirty flag since we fetched from PDS
	s.cache.SetBrews(s.sessionID, brews)
	s.cache.ClearDirty(s.sessionID, NSIDBrew)

	return brews, nil
}

func (s *AtprotoStore) UpdateBrewByRKey(ctx context.Context, rkey string, brew *models.CreateBrewRequest) error {
	// Build AT-URI references from rkeys
	if brew.BeanRKey == "" {
		return fmt.Errorf("bean_rkey is required")
	}

	beanURI := BuildATURI(s.did.String(), NSIDBean, brew.BeanRKey)

	var grinderURI, brewerURI, recipeURI string
	if brew.GrinderRKey != "" {
		grinderURI = BuildATURI(s.did.String(), NSIDGrinder, brew.GrinderRKey)
	}
	if brew.BrewerRKey != "" {
		brewerURI = BuildATURI(s.did.String(), NSIDBrewer, brew.BrewerRKey)
	}
	if brew.RecipeRKey != "" {
		recipeOwner := s.did.String()
		if brew.RecipeOwnerDID != "" {
			recipeOwner = brew.RecipeOwnerDID
		}
		recipeURI = BuildATURI(recipeOwner, NSIDRecipe, brew.RecipeRKey)
	}

	// Get the existing record to preserve createdAt
	existing, err := s.GetBrewByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("failed to get existing brew: %w", err)
	}

	// Convert to models.Brew, preserving original creation time
	brewModel := brewModelFromRequest(brew, existing.CreatedAt)

	// Convert to atproto record
	record, err := BrewToRecord(brewModel, beanURI, grinderURI, brewerURI, recipeURI)
	if err != nil {
		return fmt.Errorf("failed to convert brew to record: %w", err)
	}

	// Update record in PDS
	err = s.client.PutRecord(ctx, s.did, s.sessionID, &PutRecordInput{
		Collection: NSIDBrew,
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update brew record: %w", err)
	}

	s.writeThroughWitness(NSIDBrew, rkey, "", record)
	s.cache.InvalidateBrews(s.sessionID)

	return nil
}

func (s *AtprotoStore) DeleteBrewByRKey(ctx context.Context, rkey string) error {
	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: NSIDBrew,
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete brew record: %w", err)
	}

	s.deleteFromWitness(NSIDBrew, rkey)
	s.cache.InvalidateBrews(s.sessionID)

	return nil
}

// BeanRecord contains a bean with its AT Protocol metadata
type BeanRecord struct {
	Bean *models.Bean
	URI  string
	CID  string
}

// GetBeanRecordByRKey fetches a bean by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetBeanRecordByRKey(ctx context.Context, rkey string) (*BeanRecord, error) {
	// Try witness cache
	if wr := s.getFromWitness(ctx, NSIDBean, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			bean, err := RecordToBean(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("bean").Inc()
				bean.RKey = rkey
				if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
					if c, err := ResolveATURI(roasterRef); err == nil {
						bean.RoasterRKey = c.RKey
					}
					if roasterWR := s.getWitnessRecordByURI(ctx, roasterRef); roasterWR != nil {
						if roasterMap, err := witnessRecordToMap(roasterWR); err == nil {
							if roaster, err := RecordToRoaster(roasterMap, roasterWR.URI); err == nil {
								roaster.RKey = roasterWR.RKey
								bean.Roaster = roaster
							}
						}
					}
				}
				return &BeanRecord{
					Bean: bean,
					URI:  wr.URI,
					CID:  wr.CID,
				}, nil
			}
		}
		log.Warn().Err(err).Str("rkey", rkey).Msg("witness: failed to convert bean record, falling back to PDS")
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("bean").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDBean,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get bean record: %w", err)
	}

	atURI := BuildATURI(s.did.String(), NSIDBean, rkey)
	bean, err := RecordToBean(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert bean record: %w", err)
	}

	bean.RKey = rkey

	// Resolve roaster reference if present
	if roasterRef, ok := output.Value["roasterRef"].(string); ok && roasterRef != "" {
		if components, err := ResolveATURI(roasterRef); err == nil {
			bean.RoasterRKey = components.RKey
		}
		if len(roasterRef) > 10 && (roasterRef[:5] == "at://" || roasterRef[:4] == "did:") {
			bean.Roaster, err = ResolveRoasterRef(ctx, s.client, roasterRef, s.sessionID)
			if err != nil {
				log.Warn().Err(err).Str("bean_rkey", rkey).Str("roaster_ref", roasterRef).Msg("Failed to resolve roaster reference")
			}
		}
	}

	return &BeanRecord{
		Bean: bean,
		URI:  output.URI,
		CID:  output.CID,
	}, nil
}

// RoasterRecord contains a roaster with its AT Protocol metadata
type RoasterRecord struct {
	Roaster *models.Roaster
	URI     string
	CID     string
}

// GetRoasterRecordByRKey fetches a roaster by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetRoasterRecordByRKey(ctx context.Context, rkey string) (*RoasterRecord, error) {
	if wr := s.getFromWitness(ctx, NSIDRoaster, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			roaster, err := RecordToRoaster(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("roaster").Inc()
				roaster.RKey = rkey
				return &RoasterRecord{Roaster: roaster, URI: wr.URI, CID: wr.CID}, nil
			}
		}
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("roaster").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDRoaster,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get roaster record: %w", err)
	}

	atURI := BuildATURI(s.did.String(), NSIDRoaster, rkey)
	roaster, err := RecordToRoaster(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert roaster record: %w", err)
	}

	roaster.RKey = rkey

	return &RoasterRecord{
		Roaster: roaster,
		URI:     output.URI,
		CID:     output.CID,
	}, nil
}

// GrinderRecord contains a grinder with its AT Protocol metadata
type GrinderRecord struct {
	Grinder *models.Grinder
	URI     string
	CID     string
}

// GetGrinderRecordByRKey fetches a grinder by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetGrinderRecordByRKey(ctx context.Context, rkey string) (*GrinderRecord, error) {
	if wr := s.getFromWitness(ctx, NSIDGrinder, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			grinder, err := RecordToGrinder(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("grinder").Inc()
				grinder.RKey = rkey
				return &GrinderRecord{Grinder: grinder, URI: wr.URI, CID: wr.CID}, nil
			}
		}
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("grinder").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDGrinder,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get grinder record: %w", err)
	}

	atURI := BuildATURI(s.did.String(), NSIDGrinder, rkey)
	grinder, err := RecordToGrinder(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert grinder record: %w", err)
	}

	grinder.RKey = rkey

	return &GrinderRecord{
		Grinder: grinder,
		URI:     output.URI,
		CID:     output.CID,
	}, nil
}

// BrewerRecord contains a brewer with its AT Protocol metadata
type BrewerRecord struct {
	Brewer *models.Brewer
	URI    string
	CID    string
}

// GetBrewerRecordByRKey fetches a brewer by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetBrewerRecordByRKey(ctx context.Context, rkey string) (*BrewerRecord, error) {
	if wr := s.getFromWitness(ctx, NSIDBrewer, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			brewer, err := RecordToBrewer(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("brewer").Inc()
				brewer.RKey = rkey
				return &BrewerRecord{Brewer: brewer, URI: wr.URI, CID: wr.CID}, nil
			}
		}
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("brewer").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDBrewer,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brewer record: %w", err)
	}

	atURI := BuildATURI(s.did.String(), NSIDBrewer, rkey)
	brewer, err := RecordToBrewer(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert brewer record: %w", err)
	}

	brewer.RKey = rkey

	return &BrewerRecord{
		Brewer: brewer,
		URI:    output.URI,
		CID:    output.CID,
	}, nil
}

// ========== Bean Operations ==========

func (s *AtprotoStore) CreateBean(ctx context.Context, bean *models.CreateBeanRequest) (*models.Bean, error) {
	var roasterURI string
	if bean.RoasterRKey != "" {
		roasterURI = BuildATURI(s.did.String(), NSIDRoaster, bean.RoasterRKey)
	}

	beanModel := &models.Bean{
		Name:        bean.Name,
		Origin:      bean.Origin,
		Variety:     bean.Variety,
		RoastLevel:  bean.RoastLevel,
		Process:     bean.Process,
		Description: bean.Description,
		RoasterRKey: bean.RoasterRKey,
		Rating:      bean.Rating,
		SourceRef:   bean.SourceRef,
		CreatedAt:   time.Now().UTC(),
	}

	record, err := BeanToRecord(beanModel, roasterURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert bean to record: %w", err)
	}

	output, err := s.client.CreateRecord(ctx, s.did, s.sessionID, &CreateRecordInput{
		Collection: NSIDBean,
		Record:     record,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bean record: %w", err)
	}

	atURI, err := syntax.ParseATURI(output.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned AT-URI: %w", err)
	}

	// Store the rkey in the model
	rkey := atURI.RecordKey().String()
	beanModel.RKey = rkey

	s.writeThroughWitness(NSIDBean, rkey, output.CID, record)
	s.cache.InvalidateBeans(s.sessionID)

	return beanModel, nil
}

func (s *AtprotoStore) GetBeanByRKey(ctx context.Context, rkey string) (*models.Bean, error) {
	// Try witness cache
	if wr := s.getFromWitness(ctx, NSIDBean, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			bean, err := RecordToBean(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("bean").Inc()
				bean.RKey = rkey
				if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
					if c, err := ResolveATURI(roasterRef); err == nil {
						bean.RoasterRKey = c.RKey
					}
					// Resolve roaster from witness cache
					if roasterWR := s.getWitnessRecordByURI(ctx, roasterRef); roasterWR != nil {
						if roasterMap, err := witnessRecordToMap(roasterWR); err == nil {
							if roaster, err := RecordToRoaster(roasterMap, roasterWR.URI); err == nil {
								roaster.RKey = roasterWR.RKey
								bean.Roaster = roaster
							}
						}
					}
				}
				return bean, nil
			}
		}
		log.Warn().Err(err).Str("rkey", rkey).Msg("witness: failed to convert bean, falling back to PDS")
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("bean").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDBean,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get bean record: %w", err)
	}

	atURI := BuildATURI(s.did.String(), NSIDBean, rkey)
	bean, err := RecordToBean(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert bean record: %w", err)
	}

	bean.RKey = rkey

	// Resolve roaster reference if present
	if roasterRef, ok := output.Value["roasterRef"].(string); ok && roasterRef != "" {
		// Extract rkey from roaster ref
		if components, err := ResolveATURI(roasterRef); err == nil {
			bean.RoasterRKey = components.RKey
		}
		// Only try to resolve if it looks like a valid AT-URI
		if len(roasterRef) > 10 && (roasterRef[:5] == "at://" || roasterRef[:4] == "did:") {
			bean.Roaster, err = ResolveRoasterRef(ctx, s.client, roasterRef, s.sessionID)
			if err != nil {
				log.Warn().Err(err).Str("bean_rkey", rkey).Str("roaster_ref", roasterRef).Msg("Failed to resolve roaster reference")
			}
		}
	}

	return bean, nil
}

func (s *AtprotoStore) ListBeans(ctx context.Context) ([]*models.Bean, error) {
	// Check cache first
	userCache := s.cache.Get(s.sessionID)
	if userCache != nil && userCache.Beans() != nil && userCache.IsValid() {
		return userCache.Beans(), nil
	}

	// Try witness cache
	if wRecords := s.listFromWitness(ctx, NSIDBean); wRecords != nil {
		metrics.WitnessCacheHitsTotal.WithLabelValues("bean").Inc()
		beans := make([]*models.Bean, 0, len(wRecords))
		for _, wr := range wRecords {
			m, err := witnessRecordToMap(wr)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to parse bean")
				continue
			}
			bean, err := RecordToBean(m, wr.URI)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to convert bean")
				continue
			}
			bean.RKey = wr.RKey
			if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
				if c, err := ResolveATURI(roasterRef); err == nil {
					bean.RoasterRKey = c.RKey
				}
			}
			beans = append(beans, bean)
		}
		s.cache.SetBeans(s.sessionID, beans)
		return beans, nil
	}

	metrics.WitnessCacheMissesTotal.WithLabelValues("bean").Inc()

	// Use ListAllRecords to handle pagination automatically
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, NSIDBean)
	if err != nil {
		return nil, fmt.Errorf("failed to list bean records: %w", err)
	}

	beans := make([]*models.Bean, 0, len(output.Records))

	for _, rec := range output.Records {
		bean, err := RecordToBean(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert bean record")
			continue
		}

		// Extract rkey from URI
		if components, err := ResolveATURI(rec.URI); err == nil {
			bean.RKey = components.RKey
		}

		// Extract roaster rkey from reference (but don't fetch it - avoids N+1)
		// The caller can link roasters using LinkBeansToRoasters after fetching both
		if roasterRef, ok := rec.Value["roasterRef"].(string); ok && roasterRef != "" {
			if components, err := ResolveATURI(roasterRef); err == nil {
				bean.RoasterRKey = components.RKey
			}
		}

		beans = append(beans, bean)
	}

	// Update cache and clear dirty flag since we fetched from PDS
	s.cache.SetBeans(s.sessionID, beans)
	s.cache.ClearDirty(s.sessionID, NSIDBean)

	return beans, nil
}

// LinkBeansToRoasters populates the Roaster field on beans using a pre-fetched roasters map
// This avoids N+1 queries when listing beans with their roasters
func LinkBeansToRoasters(beans []*models.Bean, roasters []*models.Roaster) {
	// Build a map of rkey -> roaster for O(1) lookups
	roasterMap := make(map[string]*models.Roaster, len(roasters))
	for _, r := range roasters {
		roasterMap[r.RKey] = r
	}

	// Link beans to their roasters
	for _, bean := range beans {
		if bean.RoasterRKey != "" {
			bean.Roaster = roasterMap[bean.RoasterRKey]
		}
	}
}

func (s *AtprotoStore) UpdateBeanByRKey(ctx context.Context, rkey string, bean *models.UpdateBeanRequest) error {
	// Get existing to preserve createdAt
	existing, err := s.GetBeanByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("failed to get existing bean: %w", err)
	}

	var roasterURI string
	if bean.RoasterRKey != "" {
		roasterURI = BuildATURI(s.did.String(), NSIDRoaster, bean.RoasterRKey)
	}

	beanModel := &models.Bean{
		Name:        bean.Name,
		Origin:      bean.Origin,
		Variety:     bean.Variety,
		RoastLevel:  bean.RoastLevel,
		Process:     bean.Process,
		Description: bean.Description,
		RoasterRKey: bean.RoasterRKey,
		Rating:      bean.Rating,
		Closed:      bean.Closed,
		SourceRef:   bean.SourceRef,
		CreatedAt:   existing.CreatedAt,
	}

	record, err := BeanToRecord(beanModel, roasterURI)
	if err != nil {
		return fmt.Errorf("failed to convert bean to record: %w", err)
	}

	err = s.client.PutRecord(ctx, s.did, s.sessionID, &PutRecordInput{
		Collection: NSIDBean,
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update bean record: %w", err)
	}

	s.writeThroughWitness(NSIDBean, rkey, "", record)
	s.cache.InvalidateBeans(s.sessionID)

	return nil
}

func (s *AtprotoStore) DeleteBeanByRKey(ctx context.Context, rkey string) error {
	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: NSIDBean,
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete bean record: %w", err)
	}

	s.deleteFromWitness(NSIDBean, rkey)
	s.cache.InvalidateBeans(s.sessionID)

	return nil
}

// ========== Roaster Operations ==========

func (s *AtprotoStore) CreateRoaster(ctx context.Context, roaster *models.CreateRoasterRequest) (*models.Roaster, error) {
	roasterModel := &models.Roaster{
		Name:      roaster.Name,
		Location:  roaster.Location,
		Website:   roaster.Website,
		SourceRef: roaster.SourceRef,
		CreatedAt: time.Now().UTC(),
	}

	record, err := RoasterToRecord(roasterModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert roaster to record: %w", err)
	}

	output, err := s.client.CreateRecord(ctx, s.did, s.sessionID, &CreateRecordInput{
		Collection: NSIDRoaster,
		Record:     record,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create roaster record: %w", err)
	}

	atURI, err := syntax.ParseATURI(output.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned AT-URI: %w", err)
	}

	// Store the rkey in the model
	rkey := atURI.RecordKey().String()
	roasterModel.RKey = rkey

	s.writeThroughWitness(NSIDRoaster, rkey, output.CID, record)
	s.cache.InvalidateRoasters(s.sessionID)

	return roasterModel, nil
}

func (s *AtprotoStore) GetRoasterByRKey(ctx context.Context, rkey string) (*models.Roaster, error) {
	// Try witness cache
	if wr := s.getFromWitness(ctx, NSIDRoaster, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			roaster, err := RecordToRoaster(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("roaster").Inc()
				roaster.RKey = rkey
				return roaster, nil
			}
		}
		log.Warn().Err(err).Str("rkey", rkey).Msg("witness: failed to convert roaster, falling back to PDS")
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("roaster").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDRoaster,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get roaster record: %w", err)
	}

	atURI := BuildATURI(s.did.String(), NSIDRoaster, rkey)
	roaster, err := RecordToRoaster(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert roaster record: %w", err)
	}

	roaster.RKey = rkey

	return roaster, nil
}

func (s *AtprotoStore) ListRoasters(ctx context.Context) ([]*models.Roaster, error) {
	// Check cache first
	userCache := s.cache.Get(s.sessionID)
	if userCache != nil && userCache.Roasters() != nil && userCache.IsValid() {
		return userCache.Roasters(), nil
	}

	// Try witness cache
	if wRecords := s.listFromWitness(ctx, NSIDRoaster); wRecords != nil {
		metrics.WitnessCacheHitsTotal.WithLabelValues("roaster").Inc()
		roasters := make([]*models.Roaster, 0, len(wRecords))
		for _, wr := range wRecords {
			m, err := witnessRecordToMap(wr)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to parse roaster")
				continue
			}
			roaster, err := RecordToRoaster(m, wr.URI)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to convert roaster")
				continue
			}
			roaster.RKey = wr.RKey
			roasters = append(roasters, roaster)
		}
		s.cache.SetRoasters(s.sessionID, roasters)
		return roasters, nil
	}

	metrics.WitnessCacheMissesTotal.WithLabelValues("roaster").Inc()

	// Use ListAllRecords to handle pagination automatically
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, NSIDRoaster)
	if err != nil {
		return nil, fmt.Errorf("failed to list roaster records: %w", err)
	}

	roasters := make([]*models.Roaster, 0, len(output.Records))

	for _, rec := range output.Records {
		roaster, err := RecordToRoaster(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert roaster record")
			continue
		}

		// Extract rkey from URI
		if components, err := ResolveATURI(rec.URI); err == nil {
			roaster.RKey = components.RKey
		}

		roasters = append(roasters, roaster)
	}

	// Update cache and clear dirty flag since we fetched from PDS
	s.cache.SetRoasters(s.sessionID, roasters)
	s.cache.ClearDirty(s.sessionID, NSIDRoaster)

	return roasters, nil
}

func (s *AtprotoStore) UpdateRoasterByRKey(ctx context.Context, rkey string, roaster *models.UpdateRoasterRequest) error {
	// Get existing to preserve createdAt
	existing, err := s.GetRoasterByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("failed to get existing roaster: %w", err)
	}

	roasterModel := &models.Roaster{
		Name:      roaster.Name,
		Location:  roaster.Location,
		Website:   roaster.Website,
		SourceRef: roaster.SourceRef,
		CreatedAt: existing.CreatedAt,
	}

	record, err := RoasterToRecord(roasterModel)
	if err != nil {
		return fmt.Errorf("failed to convert roaster to record: %w", err)
	}

	err = s.client.PutRecord(ctx, s.did, s.sessionID, &PutRecordInput{
		Collection: NSIDRoaster,
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update roaster record: %w", err)
	}

	s.writeThroughWitness(NSIDRoaster, rkey, "", record)
	s.cache.InvalidateRoasters(s.sessionID)

	return nil
}

func (s *AtprotoStore) DeleteRoasterByRKey(ctx context.Context, rkey string) error {
	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: NSIDRoaster,
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete roaster record: %w", err)
	}

	s.deleteFromWitness(NSIDRoaster, rkey)
	s.cache.InvalidateRoasters(s.sessionID)

	return nil
}

// ========== Grinder Operations ==========

func (s *AtprotoStore) CreateGrinder(ctx context.Context, grinder *models.CreateGrinderRequest) (*models.Grinder, error) {
	grinderModel := &models.Grinder{
		Name:        grinder.Name,
		GrinderType: grinder.GrinderType,
		BurrType:    grinder.BurrType,
		Notes:       grinder.Notes,
		SourceRef:   grinder.SourceRef,
		CreatedAt:   time.Now().UTC(),
	}

	record, err := GrinderToRecord(grinderModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert grinder to record: %w", err)
	}

	output, err := s.client.CreateRecord(ctx, s.did, s.sessionID, &CreateRecordInput{
		Collection: NSIDGrinder,
		Record:     record,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create grinder record: %w", err)
	}

	atURI, err := syntax.ParseATURI(output.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned AT-URI: %w", err)
	}

	// Store the rkey in the model
	rkey := atURI.RecordKey().String()
	grinderModel.RKey = rkey

	s.writeThroughWitness(NSIDGrinder, rkey, output.CID, record)
	s.cache.InvalidateGrinders(s.sessionID)

	return grinderModel, nil
}

func (s *AtprotoStore) GetGrinderByRKey(ctx context.Context, rkey string) (*models.Grinder, error) {
	// Try witness cache
	if wr := s.getFromWitness(ctx, NSIDGrinder, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			grinder, err := RecordToGrinder(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("grinder").Inc()
				grinder.RKey = rkey
				return grinder, nil
			}
		}
		log.Warn().Err(err).Str("rkey", rkey).Msg("witness: failed to convert grinder, falling back to PDS")
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("grinder").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDGrinder,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get grinder record: %w", err)
	}

	atURI := BuildATURI(s.did.String(), NSIDGrinder, rkey)
	grinder, err := RecordToGrinder(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert grinder record: %w", err)
	}

	grinder.RKey = rkey

	return grinder, nil
}

func (s *AtprotoStore) ListGrinders(ctx context.Context) ([]*models.Grinder, error) {
	// Check cache first
	userCache := s.cache.Get(s.sessionID)
	if userCache != nil && userCache.Grinders() != nil && userCache.IsValid() {
		return userCache.Grinders(), nil
	}

	// Try witness cache
	if wRecords := s.listFromWitness(ctx, NSIDGrinder); wRecords != nil {
		metrics.WitnessCacheHitsTotal.WithLabelValues("grinder").Inc()
		grinders := make([]*models.Grinder, 0, len(wRecords))
		for _, wr := range wRecords {
			m, err := witnessRecordToMap(wr)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to parse grinder")
				continue
			}
			grinder, err := RecordToGrinder(m, wr.URI)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to convert grinder")
				continue
			}
			grinder.RKey = wr.RKey
			grinders = append(grinders, grinder)
		}
		s.cache.SetGrinders(s.sessionID, grinders)
		return grinders, nil
	}

	metrics.WitnessCacheMissesTotal.WithLabelValues("grinder").Inc()

	// Use ListAllRecords to handle pagination automatically
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, NSIDGrinder)
	if err != nil {
		return nil, fmt.Errorf("failed to list grinder records: %w", err)
	}

	grinders := make([]*models.Grinder, 0, len(output.Records))

	for _, rec := range output.Records {
		grinder, err := RecordToGrinder(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert grinder record")
			continue
		}

		// Extract rkey from URI
		if components, err := ResolveATURI(rec.URI); err == nil {
			grinder.RKey = components.RKey
		}

		grinders = append(grinders, grinder)
	}

	// Update cache and clear dirty flag since we fetched from PDS
	s.cache.SetGrinders(s.sessionID, grinders)
	s.cache.ClearDirty(s.sessionID, NSIDGrinder)

	return grinders, nil
}

func (s *AtprotoStore) UpdateGrinderByRKey(ctx context.Context, rkey string, grinder *models.UpdateGrinderRequest) error {
	// Get existing to preserve createdAt
	existing, err := s.GetGrinderByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("failed to get existing grinder: %w", err)
	}

	grinderModel := &models.Grinder{
		Name:        grinder.Name,
		GrinderType: grinder.GrinderType,
		BurrType:    grinder.BurrType,
		Notes:       grinder.Notes,
		SourceRef:   grinder.SourceRef,
		CreatedAt:   existing.CreatedAt,
	}

	record, err := GrinderToRecord(grinderModel)
	if err != nil {
		return fmt.Errorf("failed to convert grinder to record: %w", err)
	}

	err = s.client.PutRecord(ctx, s.did, s.sessionID, &PutRecordInput{
		Collection: NSIDGrinder,
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update grinder record: %w", err)
	}

	s.writeThroughWitness(NSIDGrinder, rkey, "", record)
	s.cache.InvalidateGrinders(s.sessionID)

	return nil
}

func (s *AtprotoStore) DeleteGrinderByRKey(ctx context.Context, rkey string) error {
	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: NSIDGrinder,
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete grinder record: %w", err)
	}

	s.deleteFromWitness(NSIDGrinder, rkey)
	s.cache.InvalidateGrinders(s.sessionID)

	return nil
}

// ========== Brewer Operations ==========

func (s *AtprotoStore) CreateBrewer(ctx context.Context, brewer *models.CreateBrewerRequest) (*models.Brewer, error) {
	brewerModel := &models.Brewer{
		Name:        brewer.Name,
		BrewerType:  brewer.BrewerType,
		Description: brewer.Description,
		SourceRef:   brewer.SourceRef,
		CreatedAt:   time.Now().UTC(),
	}

	record, err := BrewerToRecord(brewerModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert brewer to record: %w", err)
	}

	output, err := s.client.CreateRecord(ctx, s.did, s.sessionID, &CreateRecordInput{
		Collection: NSIDBrewer,
		Record:     record,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create brewer record: %w", err)
	}

	atURI, err := syntax.ParseATURI(output.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned AT-URI: %w", err)
	}

	// Store the rkey in the model
	rkey := atURI.RecordKey().String()
	brewerModel.RKey = rkey

	s.writeThroughWitness(NSIDBrewer, rkey, output.CID, record)
	s.cache.InvalidateBrewers(s.sessionID)

	return brewerModel, nil
}

func (s *AtprotoStore) GetBrewerByRKey(ctx context.Context, rkey string) (*models.Brewer, error) {
	// Try witness cache
	if wr := s.getFromWitness(ctx, NSIDBrewer, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			brewer, err := RecordToBrewer(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("brewer").Inc()
				brewer.RKey = rkey
				return brewer, nil
			}
		}
		log.Warn().Err(err).Str("rkey", rkey).Msg("witness: failed to convert brewer, falling back to PDS")
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("brewer").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDBrewer,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brewer record: %w", err)
	}

	atURI := BuildATURI(s.did.String(), NSIDBrewer, rkey)
	brewer, err := RecordToBrewer(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert brewer record: %w", err)
	}

	brewer.RKey = rkey

	return brewer, nil
}

func (s *AtprotoStore) ListBrewers(ctx context.Context) ([]*models.Brewer, error) {
	// Check cache first
	userCache := s.cache.Get(s.sessionID)
	if userCache != nil && userCache.Brewers() != nil && userCache.IsValid() {
		return userCache.Brewers(), nil
	}

	// Try witness cache
	if wRecords := s.listFromWitness(ctx, NSIDBrewer); wRecords != nil {
		metrics.WitnessCacheHitsTotal.WithLabelValues("brewer").Inc()
		brewers := make([]*models.Brewer, 0, len(wRecords))
		for _, wr := range wRecords {
			m, err := witnessRecordToMap(wr)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to parse brewer")
				continue
			}
			brewer, err := RecordToBrewer(m, wr.URI)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to convert brewer")
				continue
			}
			brewer.RKey = wr.RKey
			brewers = append(brewers, brewer)
		}
		s.cache.SetBrewers(s.sessionID, brewers)
		return brewers, nil
	}

	metrics.WitnessCacheMissesTotal.WithLabelValues("brewer").Inc()

	// Use ListAllRecords to handle pagination automatically
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, NSIDBrewer)
	if err != nil {
		return nil, fmt.Errorf("failed to list brewer records: %w", err)
	}

	brewers := make([]*models.Brewer, 0, len(output.Records))

	for _, rec := range output.Records {
		brewer, err := RecordToBrewer(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert brewer record")
			continue
		}

		// Extract rkey from URI
		if components, err := ResolveATURI(rec.URI); err == nil {
			brewer.RKey = components.RKey
		}

		brewers = append(brewers, brewer)
	}

	// Update cache and clear dirty flag since we fetched from PDS
	s.cache.SetBrewers(s.sessionID, brewers)
	s.cache.ClearDirty(s.sessionID, NSIDBrewer)

	return brewers, nil
}

func (s *AtprotoStore) UpdateBrewerByRKey(ctx context.Context, rkey string, brewer *models.UpdateBrewerRequest) error {
	// Get existing to preserve createdAt
	existing, err := s.GetBrewerByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("failed to get existing brewer: %w", err)
	}

	brewerModel := &models.Brewer{
		Name:        brewer.Name,
		BrewerType:  brewer.BrewerType,
		Description: brewer.Description,
		SourceRef:   brewer.SourceRef,
		CreatedAt:   existing.CreatedAt,
	}

	record, err := BrewerToRecord(brewerModel)
	if err != nil {
		return fmt.Errorf("failed to convert brewer to record: %w", err)
	}

	err = s.client.PutRecord(ctx, s.did, s.sessionID, &PutRecordInput{
		Collection: NSIDBrewer,
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update brewer record: %w", err)
	}

	s.writeThroughWitness(NSIDBrewer, rkey, "", record)
	s.cache.InvalidateBrewers(s.sessionID)

	return nil
}

func (s *AtprotoStore) DeleteBrewerByRKey(ctx context.Context, rkey string) error {
	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: NSIDBrewer,
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete brewer record: %w", err)
	}

	s.deleteFromWitness(NSIDBrewer, rkey)
	s.cache.InvalidateBrewers(s.sessionID)

	return nil
}

// ========== Recipe Operations ==========

// RecipeRecord contains a recipe with its AT Protocol metadata
type RecipeRecord struct {
	Recipe *models.Recipe
	URI    string
	CID    string
}

func (s *AtprotoStore) CreateRecipe(ctx context.Context, req *models.CreateRecipeRequest) (*models.Recipe, error) {
	var brewerURI string
	if req.BrewerRKey != "" {
		brewerURI = BuildATURI(s.did.String(), NSIDBrewer, req.BrewerRKey)
	}

	recipeModel := &models.Recipe{
		Name:         req.Name,
		BrewerRKey:   req.BrewerRKey,
		BrewerType:   req.BrewerType,
		CoffeeAmount: req.CoffeeAmount,
		WaterAmount:  req.WaterAmount,
		Notes:        req.Notes,
		SourceRef:    req.SourceRef,
		CreatedAt:    time.Now().UTC(),
	}
	if len(req.Pours) > 0 {
		recipeModel.Pours = make([]*models.Pour, len(req.Pours))
		for i, pour := range req.Pours {
			recipeModel.Pours[i] = &models.Pour{
				WaterAmount: pour.WaterAmount,
				TimeSeconds: pour.TimeSeconds,
			}
		}
	}

	record, err := RecipeToRecord(recipeModel, brewerURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert recipe to record: %w", err)
	}

	output, err := s.client.CreateRecord(ctx, s.did, s.sessionID, &CreateRecordInput{
		Collection: NSIDRecipe,
		Record:     record,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create recipe record: %w", err)
	}

	atURI, err := syntax.ParseATURI(output.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned AT-URI: %w", err)
	}

	recipeModel.RKey = atURI.RecordKey().String()

	s.writeThroughWitness(NSIDRecipe, recipeModel.RKey, output.CID, record)
	s.cache.InvalidateRecipes(s.sessionID)

	return recipeModel, nil
}

func (s *AtprotoStore) GetRecipeByRKey(ctx context.Context, rkey string) (*models.Recipe, error) {
	// Try witness cache
	if wr := s.getFromWitness(ctx, NSIDRecipe, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			recipe, err := RecordToRecipe(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("recipe").Inc()
				recipe.RKey = rkey
				if brewerRef, ok := m["brewerRef"].(string); ok && brewerRef != "" {
					if c, err := ResolveATURI(brewerRef); err == nil {
						recipe.BrewerRKey = c.RKey
					}
					if brewerWR := s.getWitnessRecordByURI(ctx, brewerRef); brewerWR != nil {
						if brewerMap, err := witnessRecordToMap(brewerWR); err == nil {
							if brewer, err := RecordToBrewer(brewerMap, brewerWR.URI); err == nil {
								brewer.RKey = brewerWR.RKey
								recipe.BrewerObj = brewer
							}
						}
					}
				}
				return recipe, nil
			}
		}
		log.Warn().Err(err).Str("rkey", rkey).Msg("witness: failed to convert recipe, falling back to PDS")
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("recipe").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDRecipe,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe record: %w", err)
	}

	atURI := BuildATURI(s.did.String(), NSIDRecipe, rkey)
	recipe, err := RecordToRecipe(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert recipe record: %w", err)
	}

	recipe.RKey = rkey

	// Resolve brewer reference if present
	if brewerRef, ok := output.Value["brewerRef"].(string); ok && brewerRef != "" {
		if components, err := ResolveATURI(brewerRef); err == nil {
			recipe.BrewerRKey = components.RKey
		}
		recipe.BrewerObj, err = ResolveBrewerRef(ctx, s.client, brewerRef, s.sessionID)
		if err != nil {
			log.Warn().Err(err).Str("recipe_rkey", rkey).Msg("Failed to resolve brewer reference")
		}
	}

	return recipe, nil
}

// GetRecipeRecordByRKey fetches a recipe by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetRecipeRecordByRKey(ctx context.Context, rkey string) (*RecipeRecord, error) {
	// Try witness cache
	if wr := s.getFromWitness(ctx, NSIDRecipe, rkey); wr != nil {
		m, err := witnessRecordToMap(wr)
		if err == nil {
			recipe, err := RecordToRecipe(m, wr.URI)
			if err == nil {
				metrics.WitnessCacheHitsTotal.WithLabelValues("recipe").Inc()
				recipe.RKey = rkey
				if brewerRef, ok := m["brewerRef"].(string); ok && brewerRef != "" {
					if c, err := ResolveATURI(brewerRef); err == nil {
						recipe.BrewerRKey = c.RKey
					}
					if brewerWR := s.getWitnessRecordByURI(ctx, brewerRef); brewerWR != nil {
						if brewerMap, err := witnessRecordToMap(brewerWR); err == nil {
							if brewer, err := RecordToBrewer(brewerMap, brewerWR.URI); err == nil {
								brewer.RKey = brewerWR.RKey
								recipe.BrewerObj = brewer
							}
						}
					}
				}
				return &RecipeRecord{
					Recipe: recipe,
					URI:    wr.URI,
					CID:    wr.CID,
				}, nil
			}
		}
		log.Warn().Err(err).Str("rkey", rkey).Msg("witness: failed to convert recipe record, falling back to PDS")
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues("recipe").Inc()
	}

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: NSIDRecipe,
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe record: %w", err)
	}

	atURI := BuildATURI(s.did.String(), NSIDRecipe, rkey)
	recipe, err := RecordToRecipe(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert recipe record: %w", err)
	}

	recipe.RKey = rkey

	if brewerRef, ok := output.Value["brewerRef"].(string); ok && brewerRef != "" {
		if components, err := ResolveATURI(brewerRef); err == nil {
			recipe.BrewerRKey = components.RKey
		}
		recipe.BrewerObj, err = ResolveBrewerRef(ctx, s.client, brewerRef, s.sessionID)
		if err != nil {
			log.Warn().Err(err).Str("recipe_rkey", rkey).Msg("Failed to resolve brewer reference")
		}
	}

	return &RecipeRecord{
		Recipe: recipe,
		URI:    output.URI,
		CID:    output.CID,
	}, nil
}

func (s *AtprotoStore) ListRecipes(ctx context.Context) ([]*models.Recipe, error) {
	// Check cache first
	userCache := s.cache.Get(s.sessionID)
	if userCache != nil && userCache.Recipes() != nil && userCache.IsValid() {
		return userCache.Recipes(), nil
	}

	// Try witness cache
	if wRecords := s.listFromWitness(ctx, NSIDRecipe); wRecords != nil {
		metrics.WitnessCacheHitsTotal.WithLabelValues("recipe").Inc()
		recipes := make([]*models.Recipe, 0, len(wRecords))
		for _, wr := range wRecords {
			m, err := witnessRecordToMap(wr)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to parse recipe")
				continue
			}
			recipe, err := RecordToRecipe(m, wr.URI)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: failed to convert recipe")
				continue
			}
			recipe.RKey = wr.RKey
			if brewerRef, ok := m["brewerRef"].(string); ok && brewerRef != "" {
				if c, err := ResolveATURI(brewerRef); err == nil {
					recipe.BrewerRKey = c.RKey
				}
			}
			recipes = append(recipes, recipe)
		}
		s.cache.SetRecipes(s.sessionID, recipes)
		return recipes, nil
	}

	metrics.WitnessCacheMissesTotal.WithLabelValues("recipe").Inc()

	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, NSIDRecipe)
	if err != nil {
		return nil, fmt.Errorf("failed to list recipe records: %w", err)
	}

	recipes := make([]*models.Recipe, 0, len(output.Records))

	for _, rec := range output.Records {
		recipe, err := RecordToRecipe(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert recipe record")
			continue
		}

		if components, err := ResolveATURI(rec.URI); err == nil {
			recipe.RKey = components.RKey
		}

		// Extract brewer rkey from reference
		if brewerRef, ok := rec.Value["brewerRef"].(string); ok && brewerRef != "" {
			if components, err := ResolveATURI(brewerRef); err == nil {
				recipe.BrewerRKey = components.RKey
			}
		}

		recipes = append(recipes, recipe)
	}

	// Clear dirty flag since we fetched from PDS
	s.cache.SetRecipes(s.sessionID, recipes)
	s.cache.ClearDirty(s.sessionID, NSIDRecipe)

	return recipes, nil
}

func (s *AtprotoStore) UpdateRecipeByRKey(ctx context.Context, rkey string, req *models.UpdateRecipeRequest) error {
	existing, err := s.GetRecipeByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("failed to get existing recipe: %w", err)
	}

	var brewerURI string
	if req.BrewerRKey != "" {
		brewerURI = BuildATURI(s.did.String(), NSIDBrewer, req.BrewerRKey)
	}

	recipeModel := &models.Recipe{
		Name:         req.Name,
		BrewerRKey:   req.BrewerRKey,
		BrewerType:   req.BrewerType,
		CoffeeAmount: req.CoffeeAmount,
		WaterAmount:  req.WaterAmount,
		Notes:        req.Notes,
		SourceRef:    existing.SourceRef,
		CreatedAt:    existing.CreatedAt,
	}
	if len(req.Pours) > 0 {
		recipeModel.Pours = make([]*models.Pour, len(req.Pours))
		for i, pour := range req.Pours {
			recipeModel.Pours[i] = &models.Pour{
				WaterAmount: pour.WaterAmount,
				TimeSeconds: pour.TimeSeconds,
			}
		}
	}

	record, err := RecipeToRecord(recipeModel, brewerURI)
	if err != nil {
		return fmt.Errorf("failed to convert recipe to record: %w", err)
	}

	err = s.client.PutRecord(ctx, s.did, s.sessionID, &PutRecordInput{
		Collection: NSIDRecipe,
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update recipe record: %w", err)
	}

	s.writeThroughWitness(NSIDRecipe, rkey, "", record)
	s.cache.InvalidateRecipes(s.sessionID)

	return nil
}

func (s *AtprotoStore) DeleteRecipeByRKey(ctx context.Context, rkey string) error {
	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: NSIDRecipe,
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete recipe record: %w", err)
	}

	s.deleteFromWitness(NSIDRecipe, rkey)
	s.cache.InvalidateRecipes(s.sessionID)

	return nil
}

// ========== Like Operations ==========

func (s *AtprotoStore) CreateLike(ctx context.Context, req *models.CreateLikeRequest) (*models.Like, error) {
	if req.SubjectURI == "" {
		return nil, fmt.Errorf("subject_uri is required")
	}
	if req.SubjectCID == "" {
		return nil, fmt.Errorf("subject_cid is required")
	}

	likeModel := &models.Like{
		SubjectURI: req.SubjectURI,
		SubjectCID: req.SubjectCID,
		CreatedAt:  time.Now().UTC(),
	}

	record, err := LikeToRecord(likeModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert like to record: %w", err)
	}

	output, err := s.client.CreateRecord(ctx, s.did, s.sessionID, &CreateRecordInput{
		Collection: NSIDLike,
		Record:     record,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create like record: %w", err)
	}

	atURI, err := syntax.ParseATURI(output.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned AT-URI: %w", err)
	}

	likeModel.RKey = atURI.RecordKey().String()

	s.writeThroughWitness(NSIDLike, likeModel.RKey, output.CID, record)

	return likeModel, nil
}

func (s *AtprotoStore) DeleteLikeByRKey(ctx context.Context, rkey string) error {
	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: NSIDLike,
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete like record: %w", err)
	}
	s.deleteFromWitness(NSIDLike, rkey)
	return nil
}

func (s *AtprotoStore) GetUserLikeForSubject(ctx context.Context, subjectURI string) (*models.Like, error) {
	// List all likes and find the one matching the subject URI
	likes, err := s.ListUserLikes(ctx)
	if err != nil {
		return nil, err
	}

	for _, like := range likes {
		if like.SubjectURI == subjectURI {
			return like, nil
		}
	}

	return nil, nil // Not found (not an error)
}

func (s *AtprotoStore) ListUserLikes(ctx context.Context) ([]*models.Like, error) {
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, NSIDLike)
	if err != nil {
		return nil, fmt.Errorf("failed to list like records: %w", err)
	}

	likes := make([]*models.Like, 0, len(output.Records))

	for _, rec := range output.Records {
		like, err := RecordToLike(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert like record")
			continue
		}

		// Extract rkey from URI
		if components, err := ResolveATURI(rec.URI); err == nil {
			like.RKey = components.RKey
		}

		likes = append(likes, like)
	}

	return likes, nil
}

// ========== Comment Operations ==========

func (s *AtprotoStore) CreateComment(ctx context.Context, req *models.CreateCommentRequest) (*models.Comment, error) {
	if req.SubjectURI == "" {
		return nil, fmt.Errorf("subject_uri is required")
	}
	if req.SubjectCID == "" {
		return nil, fmt.Errorf("subject_cid is required")
	}
	if req.Text == "" {
		return nil, fmt.Errorf("text is required")
	}

	commentModel := &models.Comment{
		SubjectURI: req.SubjectURI,
		SubjectCID: req.SubjectCID,
		Text:       req.Text,
		CreatedAt:  time.Now().UTC(),
		ParentURI:  req.ParentURI,
		ParentCID:  req.ParentCID,
	}

	record, err := CommentToRecord(commentModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert comment to record: %w", err)
	}

	output, err := s.client.CreateRecord(ctx, s.did, s.sessionID, &CreateRecordInput{
		Collection: NSIDComment,
		Record:     record,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create comment record: %w", err)
	}

	atURI, err := syntax.ParseATURI(output.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned AT-URI: %w", err)
	}

	commentModel.RKey = atURI.RecordKey().String()
	// Store the CID of this comment record (useful for threading)
	commentModel.CID = output.CID

	s.writeThroughWitness(NSIDComment, commentModel.RKey, output.CID, record)

	return commentModel, nil
}

func (s *AtprotoStore) DeleteCommentByRKey(ctx context.Context, rkey string) error {
	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: NSIDComment,
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete comment record: %w", err)
	}
	s.deleteFromWitness(NSIDComment, rkey)
	return nil
}

func (s *AtprotoStore) GetCommentsForSubject(ctx context.Context, subjectURI string) ([]*models.Comment, error) {
	// List all comments and filter by subject URI
	// Note: This is inefficient for large numbers of comments.
	// The firehose index provides a more efficient lookup.
	comments, err := s.ListUserComments(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*models.Comment
	for _, comment := range comments {
		if comment.SubjectURI == subjectURI {
			filtered = append(filtered, comment)
		}
	}

	return filtered, nil
}

func (s *AtprotoStore) ListUserComments(ctx context.Context) ([]*models.Comment, error) {
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, NSIDComment)
	if err != nil {
		return nil, fmt.Errorf("failed to list comment records: %w", err)
	}

	comments := make([]*models.Comment, 0, len(output.Records))

	for _, rec := range output.Records {
		comment, err := RecordToComment(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert comment record")
			continue
		}

		// Extract rkey from URI
		if components, err := ResolveATURI(rec.URI); err == nil {
			comment.RKey = components.RKey
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func (s *AtprotoStore) Close() error {
	// No persistent connection to close for atproto
	return nil
}
