package atproto

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tangled.org/arabica.social/arabica/internal/database"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/pdewey.com/atp"

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

	// likeNSID and commentNSID are the collection NSIDs this store reads
	// and writes for likes/comments. They default to arabica's collections
	// when unset (legacy callers / tests) but should be set per-app by
	// production callers so oolong writes to social.oolong.alpha.{like,comment}.
	likeNSID    string
	commentNSID string
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

// NewAtprotoStoreForApp builds a store wired with per-app like/comment NSIDs.
// Pass empty strings to fall back to arabica's defaults.
func NewAtprotoStoreForApp(client *Client, did syntax.DID, sessionID string, cache *SessionCache, witness WitnessCache, likeNSID, commentNSID string) database.Store {
	return &AtprotoStore{
		client:       client,
		did:          did,
		sessionID:    sessionID,
		cache:        cache,
		witnessCache: witness,
		likeNSID:     likeNSID,
		commentNSID:  commentNSID,
	}
}

func (s *AtprotoStore) likeCollection() string {
	if s.likeNSID != "" {
		return s.likeNSID
	}
	return arabica.NSIDLike
}

func (s *AtprotoStore) commentCollection() string {
	if s.commentNSID != "" {
		return s.commentNSID
	}
	return arabica.NSIDComment
}

// atpClient returns an *atp.Client scoped to this store's DID and session.
func (s *AtprotoStore) atpClient(ctx context.Context) (*atp.Client, error) {
	return s.client.AtpClient(ctx, s.did, s.sessionID)
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
	uri := atp.BuildATURI(s.did.String(), collection, rkey)
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
func (s *AtprotoStore) resolveBrewRefsFromWitness(ctx context.Context, brew *arabica.Brew, record map[string]any) {
	// Resolve bean (and its roaster)
	if beanRef, _ := record["beanRef"].(string); beanRef != "" {
		if beanWR := s.getWitnessRecordByURI(ctx, beanRef); beanWR != nil {
			if beanMap, err := witnessRecordToMap(beanWR); err == nil {
				if bean, err := arabica.RecordToBean(beanMap, beanWR.URI); err == nil {
					bean.RKey = beanWR.RKey
					// Resolve roaster ref from witness too
					if roasterRef, ok := beanMap["roasterRef"].(string); ok && roasterRef != "" {
						if rkey := atp.RKeyFromURI(roasterRef); rkey != "" {
							bean.RoasterRKey = rkey
						}
						if roasterWR := s.getWitnessRecordByURI(ctx, roasterRef); roasterWR != nil {
							if roasterMap, err := witnessRecordToMap(roasterWR); err == nil {
								if roaster, err := arabica.RecordToRoaster(roasterMap, roasterWR.URI); err == nil {
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
				if grinder, err := arabica.RecordToGrinder(grinderMap, grinderWR.URI); err == nil {
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
				if brewer, err := arabica.RecordToBrewer(brewerMap, brewerWR.URI); err == nil {
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
				if recipe, err := arabica.RecordToRecipe(recipeMap, recipeWR.URI); err == nil {
					recipe.RKey = recipeWR.RKey
					// Resolve recipe's brewer ref from witness
					if brewerRef, ok := recipeMap["brewerRef"].(string); ok && brewerRef != "" {
						if rkey := atp.RKeyFromURI(brewerRef); rkey != "" {
							recipe.BrewerRKey = rkey
						}
						if brewerWR := s.getWitnessRecordByURI(ctx, brewerRef); brewerWR != nil {
							if brewerMap, err := witnessRecordToMap(brewerWR); err == nil {
								if brewer, err := arabica.RecordToBrewer(brewerMap, brewerWR.URI); err == nil {
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
func ExtractBrewRefRKeys(brew *arabica.Brew, record map[string]any) {
	if beanRef, _ := record["beanRef"].(string); beanRef != "" {
		if rkey := atp.RKeyFromURI(beanRef); rkey != "" {
			brew.BeanRKey = rkey
		}
	}
	if grinderRef, _ := record["grinderRef"].(string); grinderRef != "" {
		if rkey := atp.RKeyFromURI(grinderRef); rkey != "" {
			brew.GrinderRKey = rkey
		}
	}
	if brewerRef, _ := record["brewerRef"].(string); brewerRef != "" {
		if rkey := atp.RKeyFromURI(brewerRef); rkey != "" {
			brew.BrewerRKey = rkey
		}
	}
	if recipeRef, _ := record["recipeRef"].(string); recipeRef != "" {
		if rkey := atp.RKeyFromURI(recipeRef); rkey != "" {
			brew.RecipeRKey = rkey
		}
	}
}

// brewModelFromRequest converts a CreateBrewRequest into a Brew model with the given creation time.
func brewModelFromRequest(req *arabica.CreateBrewRequest, createdAt time.Time) *arabica.Brew {
	brew := &arabica.Brew{
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
		brew.Pours = make([]*arabica.Pour, len(req.Pours))
		for i, pour := range req.Pours {
			brew.Pours[i] = &arabica.Pour{
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

// resolveBrewRefs dispatches reference resolution. When the brew came from
// the witness cache (fromWitness=true), all refs are resolved from witness
// only — keeps the read path PDS-free. When the brew came from PDS,
// references go to the PDS resolvers (ResolveBrewRefs handles
// bean/grinder/brewer; arabica.ResolveRecipe handles recipe). Failures are
// logged but do not fail the brew read.
func (s *AtprotoStore) resolveBrewRefs(ctx context.Context, brew *arabica.Brew, record map[string]any, fromWitness bool) {
	if fromWitness {
		s.resolveBrewRefsFromWitness(ctx, brew, record)
		return
	}
	beanRef, _ := record["beanRef"].(string)
	grinderRef, _ := record["grinderRef"].(string)
	brewerRef, _ := record["brewerRef"].(string)
	atpClient, err := s.atpClient(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("get atp client for brew refs")
		return
	}
	if err := arabica.ResolveBrewRefs(ctx, atpClient, brew, beanRef, grinderRef, brewerRef); err != nil {
		log.Warn().Err(err).Msg("resolve brew refs")
	}
	if recipeRef, _ := record["recipeRef"].(string); recipeRef != "" {
		recipe, err := arabica.ResolveRecipe(ctx, atpClient, recipeRef)
		if err != nil {
			log.Warn().Err(err).Str("ref", recipeRef).Msg("resolve recipe ref")
		} else {
			brew.RecipeObj = recipe
		}
	}
}

func (s *AtprotoStore) CreateBrew(ctx context.Context, brew *arabica.CreateBrewRequest, userID int) (*arabica.Brew, error) {
	if brew.BeanRKey == "" {
		return nil, fmt.Errorf("bean_rkey is required")
	}
	beanURI := atp.BuildATURI(s.did.String(), arabica.NSIDBean, brew.BeanRKey)
	var grinderURI, brewerURI, recipeURI string
	if brew.GrinderRKey != "" {
		grinderURI = atp.BuildATURI(s.did.String(), arabica.NSIDGrinder, brew.GrinderRKey)
	}
	if brew.BrewerRKey != "" {
		brewerURI = atp.BuildATURI(s.did.String(), arabica.NSIDBrewer, brew.BrewerRKey)
	}
	if brew.RecipeRKey != "" {
		recipeOwner := s.did.String()
		if brew.RecipeOwnerDID != "" {
			recipeOwner = brew.RecipeOwnerDID
		}
		recipeURI = atp.BuildATURI(recipeOwner, arabica.NSIDRecipe, brew.RecipeRKey)
	}

	model := brewModelFromRequest(brew, time.Now().UTC())
	record, err := arabica.BrewToRecord(model, beanURI, grinderURI, brewerURI, recipeURI)
	if err != nil {
		return nil, fmt.Errorf("convert brew: %w", err)
	}
	rkey, _, err := s.putRecord(ctx, arabica.NSIDBrew, "", record)
	if err != nil {
		return nil, err
	}
	model.RKey = rkey
	// Populate Bean/GrinderObj/BrewerObj for the response.
	if atpClient, err := s.atpClient(ctx); err == nil {
		if err := arabica.ResolveBrewRefs(ctx, atpClient, model, beanURI, grinderURI, brewerURI); err != nil {
			log.Warn().Err(err).Str("brew_rkey", rkey).Msg("resolve brew refs after create")
		}
	}
	return model, nil
}

func (s *AtprotoStore) GetBrewByRKey(ctx context.Context, rkey string) (*arabica.Brew, error) {
	rec, uri, _, hit, fromWitness, err := s.fetchRecord(ctx, arabica.NSIDBrew, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("brew %s not found", rkey)
	}
	brew, err := arabica.RecordToBrew(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert brew: %w", err)
	}
	brew.RKey = rkey
	ExtractBrewRefRKeys(brew, rec)
	s.resolveBrewRefs(ctx, brew, rec, fromWitness)
	return brew, nil
}

// BrewRecord contains a brew with its AT Protocol metadata
type BrewRecord struct {
	Brew *arabica.Brew
	URI  string
	CID  string
}

// GetBrewRecordByRKey fetches a brew by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetBrewRecordByRKey(ctx context.Context, rkey string) (*BrewRecord, error) {
	rec, uri, cid, hit, fromWitness, err := s.fetchRecord(ctx, arabica.NSIDBrew, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("brew record %s not found", rkey)
	}
	brew, err := arabica.RecordToBrew(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert brew: %w", err)
	}
	brew.RKey = rkey
	ExtractBrewRefRKeys(brew, rec)
	s.resolveBrewRefs(ctx, brew, rec, fromWitness)
	return &BrewRecord{Brew: brew, URI: uri, CID: cid}, nil
}

func (s *AtprotoStore) ListBrews(ctx context.Context, userID int, offset, limit int) ([]*arabica.Brew, error) {
	// For paginated requests, skip session cache and go directly to the witness
	// cache (local SQLite) with LIMIT/OFFSET. The session cache is only useful for
	// the "fetch all" case (e.g., export, client-side cache).
	if limit > 0 {
		return s.listBrewsPage(ctx, offset, limit)
	}

	// Non-paginated: use session cache then fetch all records.
	if uc := s.cache.Get(s.sessionID); uc != nil && uc.Brews() != nil && uc.IsValid() {
		return uc.Brews(), nil
	}
	raws, err := s.fetchAllRecords(ctx, arabica.NSIDBrew)
	if err != nil {
		return nil, err
	}
	brews := s.convertBrewRecords(raws)
	// Resolve references in bulk
	s.resolveBrewReferences(ctx, brews)
	s.cache.SetRecords(s.sessionID, arabica.NSIDBrew, brews)
	s.cache.ClearDirty(s.sessionID, arabica.NSIDBrew)
	return brews, nil
}

// listBrewsPage fetches a single page of brews from the witness cache with
// LIMIT/OFFSET, resolving references in bulk.
func (s *AtprotoStore) listBrewsPage(ctx context.Context, offset, limit int) ([]*arabica.Brew, error) {
	raws, err := s.fetchPaginatedRecords(ctx, arabica.NSIDBrew, offset, limit)
	if err != nil {
		return nil, err
	}
	brews := s.convertBrewRecords(raws)
	s.resolveBrewReferences(ctx, brews)
	return brews, nil
}

// convertBrewRecords converts raw records to typed Brew models.
func (s *AtprotoStore) convertBrewRecords(raws []rawRecord) []*arabica.Brew {
	brews := make([]*arabica.Brew, 0, len(raws))
	for _, r := range raws {
		brew, err := arabica.RecordToBrew(r.Record, r.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", r.URI).Msg("Failed to convert brew record")
			continue
		}
		brew.RKey = r.RKey
		ExtractBrewRefRKeys(brew, r.Record)
		brews = append(brews, brew)
	}
	return brews
}

// resolveBrewReferences fetches entity collections and links them to brews.
func (s *AtprotoStore) resolveBrewReferences(ctx context.Context, brews []*arabica.Brew) {
	beans, _ := s.ListBeans(ctx)
	grinders, _ := s.ListGrinders(ctx)
	brewers, _ := s.ListBrewers(ctx)
	roasters, _ := s.ListRoasters(ctx)
	recipes, _ := s.ListRecipes(ctx)
	beanMap := make(map[string]*arabica.Bean, len(beans))
	for _, b := range beans {
		beanMap[b.RKey] = b
	}
	grinderMap := make(map[string]*arabica.Grinder, len(grinders))
	for _, g := range grinders {
		grinderMap[g.RKey] = g
	}
	brewerMap := make(map[string]*arabica.Brewer, len(brewers))
	for _, b := range brewers {
		brewerMap[b.RKey] = b
	}
	roasterMap := make(map[string]*arabica.Roaster, len(roasters))
	for _, r := range roasters {
		roasterMap[r.RKey] = r
	}
	recipeMap := make(map[string]*arabica.Recipe, len(recipes))
	for _, r := range recipes {
		recipeMap[r.RKey] = r
	}
	for _, brew := range brews {
		if brew.BeanRKey != "" {
			brew.Bean = beanMap[brew.BeanRKey]
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
}

func (s *AtprotoStore) UpdateBrewByRKey(ctx context.Context, rkey string, brew *arabica.CreateBrewRequest) error {
	if brew.BeanRKey == "" {
		return fmt.Errorf("bean_rkey is required")
	}
	beanURI := atp.BuildATURI(s.did.String(), arabica.NSIDBean, brew.BeanRKey)
	var grinderURI, brewerURI, recipeURI string
	if brew.GrinderRKey != "" {
		grinderURI = atp.BuildATURI(s.did.String(), arabica.NSIDGrinder, brew.GrinderRKey)
	}
	if brew.BrewerRKey != "" {
		brewerURI = atp.BuildATURI(s.did.String(), arabica.NSIDBrewer, brew.BrewerRKey)
	}
	if brew.RecipeRKey != "" {
		recipeOwner := s.did.String()
		if brew.RecipeOwnerDID != "" {
			recipeOwner = brew.RecipeOwnerDID
		}
		recipeURI = atp.BuildATURI(recipeOwner, arabica.NSIDRecipe, brew.RecipeRKey)
	}

	existing, err := s.GetBrewByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("get existing brew: %w", err)
	}
	model := brewModelFromRequest(brew, existing.CreatedAt)
	record, err := arabica.BrewToRecord(model, beanURI, grinderURI, brewerURI, recipeURI)
	if err != nil {
		return fmt.Errorf("convert brew: %w", err)
	}
	_, _, err = s.putRecord(ctx, arabica.NSIDBrew, rkey, record)
	return err
}

func (s *AtprotoStore) DeleteBrewByRKey(ctx context.Context, rkey string) error {
	return s.removeRecord(ctx, arabica.NSIDBrew, rkey)
}

// BeanRecord contains a bean with its AT Protocol metadata
type BeanRecord struct {
	Bean *arabica.Bean
	URI  string
	CID  string
}

// GetBeanRecordByRKey fetches a bean by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetBeanRecordByRKey(ctx context.Context, rkey string) (*BeanRecord, error) {
	rec, uri, cid, hit, _, err := s.fetchRecord(ctx, arabica.NSIDBean, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("bean record %s not found", rkey)
	}
	bean, err := arabica.RecordToBean(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert bean: %w", err)
	}
	bean.RKey = rkey
	s.resolveBeanRefs(ctx, bean, rec)
	return &BeanRecord{Bean: bean, URI: uri, CID: cid}, nil
}

// RoasterRecord contains a roaster with its AT Protocol metadata
type RoasterRecord struct {
	Roaster *arabica.Roaster
	URI     string
	CID     string
}

// GetRoasterRecordByRKey fetches a roaster by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetRoasterRecordByRKey(ctx context.Context, rkey string) (*RoasterRecord, error) {
	rec, uri, cid, hit, _, err := s.fetchRecord(ctx, arabica.NSIDRoaster, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("roaster record %s not found", rkey)
	}
	roaster, err := arabica.RecordToRoaster(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert roaster: %w", err)
	}
	roaster.RKey = rkey
	return &RoasterRecord{Roaster: roaster, URI: uri, CID: cid}, nil
}

// GrinderRecord contains a grinder with its AT Protocol metadata
type GrinderRecord struct {
	Grinder *arabica.Grinder
	URI     string
	CID     string
}

// GetGrinderRecordByRKey fetches a grinder by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetGrinderRecordByRKey(ctx context.Context, rkey string) (*GrinderRecord, error) {
	rec, uri, cid, hit, _, err := s.fetchRecord(ctx, arabica.NSIDGrinder, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("grinder record %s not found", rkey)
	}
	grinder, err := arabica.RecordToGrinder(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert grinder: %w", err)
	}
	grinder.RKey = rkey
	return &GrinderRecord{Grinder: grinder, URI: uri, CID: cid}, nil
}

// BrewerRecord contains a brewer with its AT Protocol metadata
type BrewerRecord struct {
	Brewer *arabica.Brewer
	URI    string
	CID    string
}

// GetBrewerRecordByRKey fetches a brewer by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetBrewerRecordByRKey(ctx context.Context, rkey string) (*BrewerRecord, error) {
	rec, uri, cid, hit, _, err := s.fetchRecord(ctx, arabica.NSIDBrewer, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("brewer record %s not found", rkey)
	}
	brewer, err := arabica.RecordToBrewer(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert brewer: %w", err)
	}
	brewer.RKey = rkey
	return &BrewerRecord{Brewer: brewer, URI: uri, CID: cid}, nil
}

// ========== Bean Operations ==========

// resolveBeanRefs populates bean.RoasterRKey and bean.Roaster from the
// record's roasterRef field. RoasterRKey is always extracted (cheap); the
// full Roaster is resolved by trying the witness cache first, falling
// back to the PDS resolver. Failures are logged but do not fail the bean
// read.
func (s *AtprotoStore) resolveBeanRefs(ctx context.Context, bean *arabica.Bean, record map[string]any) {
	roasterRef, ok := record["roasterRef"].(string)
	if !ok || roasterRef == "" {
		return
	}
	if rkey := atp.RKeyFromURI(roasterRef); rkey != "" {
		bean.RoasterRKey = rkey
	}
	// Try witness cache first
	if wr := s.getWitnessRecordByURI(ctx, roasterRef); wr != nil {
		if m, err := witnessRecordToMap(wr); err == nil {
			if roaster, err := arabica.RecordToRoaster(m, wr.URI); err == nil {
				roaster.RKey = wr.RKey
				bean.Roaster = roaster
				return
			}
		}
	}
	// Fall back to PDS resolver
	if len(roasterRef) > 10 && (roasterRef[:5] == "at://" || roasterRef[:4] == "did:") {
		atpClient, err := s.atpClient(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("get atp client for roaster ref")
			return
		}
		roaster, err := arabica.ResolveRoaster(ctx, atpClient, roasterRef)
		if err != nil {
			log.Warn().Err(err).Str("ref", roasterRef).Msg("resolve roaster ref")
			return
		}
		bean.Roaster = roaster
	}
}

// extractBeanRoasterRKey is the cheap variant for list paths: only the
// RoasterRKey is populated; full Roaster resolution happens later via
// LinkBeansToRoasters to avoid N+1 lookups.
func extractBeanRoasterRKey(bean *arabica.Bean, record map[string]any) {
	roasterRef, ok := record["roasterRef"].(string)
	if !ok || roasterRef == "" {
		return
	}
	if rkey := atp.RKeyFromURI(roasterRef); rkey != "" {
		bean.RoasterRKey = rkey
	}
}

func (s *AtprotoStore) CreateBean(ctx context.Context, bean *arabica.CreateBeanRequest) (*arabica.Bean, error) {
	var roasterURI string
	if bean.RoasterRKey != "" {
		roasterURI = atp.BuildATURI(s.did.String(), arabica.NSIDRoaster, bean.RoasterRKey)
	}
	model := &arabica.Bean{
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
	record, err := arabica.BeanToRecord(model, roasterURI)
	if err != nil {
		return nil, fmt.Errorf("convert bean: %w", err)
	}
	rkey, _, err := s.putRecord(ctx, arabica.NSIDBean, "", record)
	if err != nil {
		return nil, err
	}
	model.RKey = rkey
	return model, nil
}

func (s *AtprotoStore) GetBeanByRKey(ctx context.Context, rkey string) (*arabica.Bean, error) {
	rec, uri, _, hit, _, err := s.fetchRecord(ctx, arabica.NSIDBean, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("bean %s not found", rkey)
	}
	bean, err := arabica.RecordToBean(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert bean: %w", err)
	}
	bean.RKey = rkey
	s.resolveBeanRefs(ctx, bean, rec)
	return bean, nil
}

func (s *AtprotoStore) ListBeans(ctx context.Context) ([]*arabica.Bean, error) {
	if uc := s.cache.Get(s.sessionID); uc != nil && uc.Beans() != nil && uc.IsValid() {
		return uc.Beans(), nil
	}
	raws, err := s.fetchAllRecords(ctx, arabica.NSIDBean)
	if err != nil {
		return nil, err
	}
	beans := make([]*arabica.Bean, 0, len(raws))
	for _, r := range raws {
		bean, err := arabica.RecordToBean(r.Record, r.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", r.URI).Msg("Failed to convert bean record")
			continue
		}
		bean.RKey = r.RKey
		// List uses cheap rkey-only ref extraction; callers run
		// LinkBeansToRoasters separately to avoid N+1 lookups.
		extractBeanRoasterRKey(bean, r.Record)
		beans = append(beans, bean)
	}
	s.cache.SetRecords(s.sessionID, arabica.NSIDBean, beans)
	s.cache.ClearDirty(s.sessionID, arabica.NSIDBean)
	return beans, nil
}

// LinkBeansToRoasters populates the Roaster field on beans using a pre-fetched roasters map
// This avoids N+1 queries when listing beans with their roasters
func LinkBeansToRoasters(beans []*arabica.Bean, roasters []*arabica.Roaster) {
	// Build a map of rkey -> roaster for O(1) lookups
	roasterMap := make(map[string]*arabica.Roaster, len(roasters))
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

func (s *AtprotoStore) UpdateBeanByRKey(ctx context.Context, rkey string, bean *arabica.UpdateBeanRequest) error {
	existing, err := s.GetBeanByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("get existing bean: %w", err)
	}
	var roasterURI string
	if bean.RoasterRKey != "" {
		roasterURI = atp.BuildATURI(s.did.String(), arabica.NSIDRoaster, bean.RoasterRKey)
	}
	model := &arabica.Bean{
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
	record, err := arabica.BeanToRecord(model, roasterURI)
	if err != nil {
		return fmt.Errorf("convert bean: %w", err)
	}
	_, _, err = s.putRecord(ctx, arabica.NSIDBean, rkey, record)
	return err
}

func (s *AtprotoStore) DeleteBeanByRKey(ctx context.Context, rkey string) error {
	return s.removeRecord(ctx, arabica.NSIDBean, rkey)
}

// ========== Roaster Operations ==========

func (s *AtprotoStore) CreateRoaster(ctx context.Context, roaster *arabica.CreateRoasterRequest) (*arabica.Roaster, error) {
	return CreateEntity(ctx, s, roasterCodec, &arabica.Roaster{
		Name:      roaster.Name,
		Location:  roaster.Location,
		Website:   roaster.Website,
		SourceRef: roaster.SourceRef,
		CreatedAt: time.Now().UTC(),
	})
}

func (s *AtprotoStore) GetRoasterByRKey(ctx context.Context, rkey string) (*arabica.Roaster, error) {
	return GetEntity(ctx, s, roasterCodec, rkey)
}

func (s *AtprotoStore) ListRoasters(ctx context.Context) ([]*arabica.Roaster, error) {
	return ListEntity(ctx, s, roasterCodec, func() []*arabica.Roaster {
		return s.cache.Get(s.sessionID).Roasters()
	})
}

func (s *AtprotoStore) UpdateRoasterByRKey(ctx context.Context, rkey string, roaster *arabica.UpdateRoasterRequest) error {
	existing, err := s.GetRoasterByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("get existing roaster: %w", err)
	}
	err = UpdateEntity(ctx, s, roasterCodec, rkey, &arabica.Roaster{
		Name:      roaster.Name,
		Location:  roaster.Location,
		Website:   roaster.Website,
		SourceRef: roaster.SourceRef,
		CreatedAt: existing.CreatedAt,
	})
	if err != nil {
		return err
	}
	// Beans denormalize roaster data; invalidate them too.
	s.cache.InvalidateRecords(s.sessionID, arabica.NSIDBean)
	return nil
}

func (s *AtprotoStore) DeleteRoasterByRKey(ctx context.Context, rkey string) error {
	if err := DeleteEntity(ctx, s, arabica.NSIDRoaster, rkey); err != nil {
		return err
	}
	// Beans denormalize roaster data; invalidate them too.
	s.cache.InvalidateRecords(s.sessionID, arabica.NSIDBean)
	return nil
}

// ========== Grinder Operations ==========

func (s *AtprotoStore) CreateGrinder(ctx context.Context, grinder *arabica.CreateGrinderRequest) (*arabica.Grinder, error) {
	return CreateEntity(ctx, s, grinderCodec, &arabica.Grinder{
		Name:        grinder.Name,
		GrinderType: grinder.GrinderType,
		BurrType:    grinder.BurrType,
		Notes:       grinder.Notes,
		SourceRef:   grinder.SourceRef,
		CreatedAt:   time.Now().UTC(),
	})
}

func (s *AtprotoStore) GetGrinderByRKey(ctx context.Context, rkey string) (*arabica.Grinder, error) {
	return GetEntity(ctx, s, grinderCodec, rkey)
}

func (s *AtprotoStore) ListGrinders(ctx context.Context) ([]*arabica.Grinder, error) {
	return ListEntity(ctx, s, grinderCodec, func() []*arabica.Grinder {
		return s.cache.Get(s.sessionID).Grinders()
	})
}

func (s *AtprotoStore) UpdateGrinderByRKey(ctx context.Context, rkey string, grinder *arabica.UpdateGrinderRequest) error {
	existing, err := s.GetGrinderByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("get existing grinder: %w", err)
	}
	model := &arabica.Grinder{
		Name:        grinder.Name,
		GrinderType: grinder.GrinderType,
		BurrType:    grinder.BurrType,
		Notes:       grinder.Notes,
		SourceRef:   grinder.SourceRef,
		CreatedAt:   existing.CreatedAt,
	}
	record, err := arabica.GrinderToRecord(model)
	if err != nil {
		return fmt.Errorf("convert grinder: %w", err)
	}
	_, _, err = s.putRecord(ctx, arabica.NSIDGrinder, rkey, record)
	return err
}

func (s *AtprotoStore) DeleteGrinderByRKey(ctx context.Context, rkey string) error {
	return s.removeRecord(ctx, arabica.NSIDGrinder, rkey)
}

// ========== Brewer Operations ==========

func (s *AtprotoStore) CreateBrewer(ctx context.Context, brewer *arabica.CreateBrewerRequest) (*arabica.Brewer, error) {
	return CreateEntity(ctx, s, brewerCodec, &arabica.Brewer{
		Name:        brewer.Name,
		BrewerType:  brewer.BrewerType,
		Description: brewer.Description,
		SourceRef:   brewer.SourceRef,
		CreatedAt:   time.Now().UTC(),
	})
}

func (s *AtprotoStore) GetBrewerByRKey(ctx context.Context, rkey string) (*arabica.Brewer, error) {
	return GetEntity(ctx, s, brewerCodec, rkey)
}

func (s *AtprotoStore) ListBrewers(ctx context.Context) ([]*arabica.Brewer, error) {
	return ListEntity(ctx, s, brewerCodec, func() []*arabica.Brewer {
		return s.cache.Get(s.sessionID).Brewers()
	})
}

func (s *AtprotoStore) UpdateBrewerByRKey(ctx context.Context, rkey string, brewer *arabica.UpdateBrewerRequest) error {
	existing, err := s.GetBrewerByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("get existing brewer: %w", err)
	}
	return UpdateEntity(ctx, s, brewerCodec, rkey, &arabica.Brewer{
		Name:        brewer.Name,
		BrewerType:  brewer.BrewerType,
		Description: brewer.Description,
		SourceRef:   brewer.SourceRef,
		CreatedAt:   existing.CreatedAt,
	})
}

func (s *AtprotoStore) DeleteBrewerByRKey(ctx context.Context, rkey string) error {
	return s.removeRecord(ctx, arabica.NSIDBrewer, rkey)
}

// ========== Recipe Operations ==========

// RecipeRecord contains a recipe with its AT Protocol metadata
type RecipeRecord struct {
	Recipe *arabica.Recipe
	URI    string
	CID    string
}

// resolveRecipeRefs populates recipe.BrewerRKey and recipe.BrewerObj from
// the record's brewerRef field. Tries the witness cache first, falls back
// to arabica.ResolveBrewer.
func (s *AtprotoStore) resolveRecipeRefs(ctx context.Context, recipe *arabica.Recipe, record map[string]any) {
	brewerRef, ok := record["brewerRef"].(string)
	if !ok || brewerRef == "" {
		return
	}
	if rkey := atp.RKeyFromURI(brewerRef); rkey != "" {
		recipe.BrewerRKey = rkey
	}
	if wr := s.getWitnessRecordByURI(ctx, brewerRef); wr != nil {
		if m, err := witnessRecordToMap(wr); err == nil {
			if brewer, err := arabica.RecordToBrewer(m, wr.URI); err == nil {
				brewer.RKey = wr.RKey
				recipe.BrewerObj = brewer
				return
			}
		}
	}
	atpClient, err := s.atpClient(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("get atp client for brewer ref")
		return
	}
	brewer, err := arabica.ResolveBrewer(ctx, atpClient, brewerRef)
	if err != nil {
		log.Warn().Err(err).Str("ref", brewerRef).Msg("resolve brewer ref")
		return
	}
	recipe.BrewerObj = brewer
}

// extractRecipeBrewerRKey populates recipe.BrewerRKey only (no full
// resolution). Used by list paths.
func extractRecipeBrewerRKey(recipe *arabica.Recipe, record map[string]any) {
	brewerRef, ok := record["brewerRef"].(string)
	if !ok || brewerRef == "" {
		return
	}
	if rkey := atp.RKeyFromURI(brewerRef); rkey != "" {
		recipe.BrewerRKey = rkey
	}
}

func recipeModelFromCreate(req *arabica.CreateRecipeRequest) *arabica.Recipe {
	model := &arabica.Recipe{
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
		model.Pours = make([]*arabica.Pour, len(req.Pours))
		for i, p := range req.Pours {
			model.Pours[i] = &arabica.Pour{WaterAmount: p.WaterAmount, TimeSeconds: p.TimeSeconds}
		}
	}
	return model
}

func (s *AtprotoStore) CreateRecipe(ctx context.Context, req *arabica.CreateRecipeRequest) (*arabica.Recipe, error) {
	var brewerURI string
	if req.BrewerRKey != "" {
		brewerURI = atp.BuildATURI(s.did.String(), arabica.NSIDBrewer, req.BrewerRKey)
	}
	model := recipeModelFromCreate(req)
	record, err := arabica.RecipeToRecord(model, brewerURI)
	if err != nil {
		return nil, fmt.Errorf("convert recipe: %w", err)
	}
	rkey, _, err := s.putRecord(ctx, arabica.NSIDRecipe, "", record)
	if err != nil {
		return nil, err
	}
	model.RKey = rkey
	return model, nil
}

func (s *AtprotoStore) GetRecipeByRKey(ctx context.Context, rkey string) (*arabica.Recipe, error) {
	rec, uri, _, hit, _, err := s.fetchRecord(ctx, arabica.NSIDRecipe, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("recipe %s not found", rkey)
	}
	recipe, err := arabica.RecordToRecipe(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert recipe: %w", err)
	}
	recipe.RKey = rkey
	s.resolveRecipeRefs(ctx, recipe, rec)
	return recipe, nil
}

// GetRecipeRecordByRKey fetches a recipe by rkey and returns it with its AT Protocol metadata
func (s *AtprotoStore) GetRecipeRecordByRKey(ctx context.Context, rkey string) (*RecipeRecord, error) {
	rec, uri, cid, hit, _, err := s.fetchRecord(ctx, arabica.NSIDRecipe, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("recipe record %s not found", rkey)
	}
	recipe, err := arabica.RecordToRecipe(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert recipe: %w", err)
	}
	recipe.RKey = rkey
	s.resolveRecipeRefs(ctx, recipe, rec)
	return &RecipeRecord{Recipe: recipe, URI: uri, CID: cid}, nil
}

func (s *AtprotoStore) ListRecipes(ctx context.Context) ([]*arabica.Recipe, error) {
	if uc := s.cache.Get(s.sessionID); uc != nil && uc.Recipes() != nil && uc.IsValid() {
		return uc.Recipes(), nil
	}
	raws, err := s.fetchAllRecords(ctx, arabica.NSIDRecipe)
	if err != nil {
		return nil, err
	}
	recipes := make([]*arabica.Recipe, 0, len(raws))
	for _, r := range raws {
		recipe, err := arabica.RecordToRecipe(r.Record, r.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", r.URI).Msg("Failed to convert recipe record")
			continue
		}
		recipe.RKey = r.RKey
		extractRecipeBrewerRKey(recipe, r.Record)
		recipes = append(recipes, recipe)
	}
	s.cache.SetRecords(s.sessionID, arabica.NSIDRecipe, recipes)
	s.cache.ClearDirty(s.sessionID, arabica.NSIDRecipe)
	return recipes, nil
}

func (s *AtprotoStore) UpdateRecipeByRKey(ctx context.Context, rkey string, req *arabica.UpdateRecipeRequest) error {
	existing, err := s.GetRecipeByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("get existing recipe: %w", err)
	}
	var brewerURI string
	if req.BrewerRKey != "" {
		brewerURI = atp.BuildATURI(s.did.String(), arabica.NSIDBrewer, req.BrewerRKey)
	}
	model := &arabica.Recipe{
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
		model.Pours = make([]*arabica.Pour, len(req.Pours))
		for i, p := range req.Pours {
			model.Pours[i] = &arabica.Pour{WaterAmount: p.WaterAmount, TimeSeconds: p.TimeSeconds}
		}
	}
	record, err := arabica.RecipeToRecord(model, brewerURI)
	if err != nil {
		return fmt.Errorf("convert recipe: %w", err)
	}
	_, _, err = s.putRecord(ctx, arabica.NSIDRecipe, rkey, record)
	return err
}

func (s *AtprotoStore) DeleteRecipeByRKey(ctx context.Context, rkey string) error {
	return s.removeRecord(ctx, arabica.NSIDRecipe, rkey)
}

// ========== Like Operations ==========

func (s *AtprotoStore) CreateLike(ctx context.Context, req *arabica.CreateLikeRequest) (*arabica.Like, error) {
	if req.SubjectURI == "" {
		return nil, fmt.Errorf("subject_uri is required")
	}
	if req.SubjectCID == "" {
		return nil, fmt.Errorf("subject_cid is required")
	}

	likeModel := &arabica.Like{
		SubjectURI: req.SubjectURI,
		SubjectCID: req.SubjectCID,
		CreatedAt:  time.Now().UTC(),
	}

	record, err := arabica.LikeToRecord(likeModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert like to record: %w", err)
	}
	collection := s.likeCollection()
	record["$type"] = collection

	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get atp client: %w", err)
	}
	uri, cid, err := atpClient.CreateRecord(ctx, collection, record)
	if err != nil {
		return nil, fmt.Errorf("failed to create like record: %w", err)
	}

	atURI, err := syntax.ParseATURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned AT-URI: %w", err)
	}

	likeModel.RKey = atURI.RecordKey().String()

	s.writeThroughWitness(collection, likeModel.RKey, cid, record)

	return likeModel, nil
}

func (s *AtprotoStore) DeleteLikeByRKey(ctx context.Context, rkey string) error {
	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return fmt.Errorf("get atp client: %w", err)
	}
	collection := s.likeCollection()
	if err := atpClient.DeleteRecord(ctx, collection, rkey); err != nil {
		return fmt.Errorf("failed to delete like record: %w", err)
	}
	s.deleteFromWitness(collection, rkey)
	return nil
}

func (s *AtprotoStore) GetUserLikeForSubject(ctx context.Context, subjectURI string) (*arabica.Like, error) {
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

func (s *AtprotoStore) ListUserLikes(ctx context.Context) ([]*arabica.Like, error) {
	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get atp client: %w", err)
	}
	records, err := atpClient.ListAllRecords(ctx, s.likeCollection())
	if err != nil {
		return nil, fmt.Errorf("failed to list like records: %w", err)
	}

	likes := make([]*arabica.Like, 0, len(records))

	for _, rec := range records {
		like, err := arabica.RecordToLike(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert like record")
			continue
		}

		// Extract rkey from URI
		if rkey := atp.RKeyFromURI(rec.URI); rkey != "" {
			like.RKey = rkey
		}

		likes = append(likes, like)
	}

	return likes, nil
}

// ========== Comment Operations ==========

func (s *AtprotoStore) CreateComment(ctx context.Context, req *arabica.CreateCommentRequest) (*arabica.Comment, error) {
	if req.SubjectURI == "" {
		return nil, fmt.Errorf("subject_uri is required")
	}
	if req.SubjectCID == "" {
		return nil, fmt.Errorf("subject_cid is required")
	}
	if req.Text == "" {
		return nil, fmt.Errorf("text is required")
	}

	commentModel := &arabica.Comment{
		SubjectURI: req.SubjectURI,
		SubjectCID: req.SubjectCID,
		Text:       req.Text,
		CreatedAt:  time.Now().UTC(),
		ParentURI:  req.ParentURI,
		ParentCID:  req.ParentCID,
	}

	record, err := arabica.CommentToRecord(commentModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert comment to record: %w", err)
	}
	collection := s.commentCollection()
	record["$type"] = collection

	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get atp client: %w", err)
	}
	uri, cid, err := atpClient.CreateRecord(ctx, collection, record)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment record: %w", err)
	}

	atURI, err := syntax.ParseATURI(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse returned AT-URI: %w", err)
	}

	commentModel.RKey = atURI.RecordKey().String()
	// Store the CID of this comment record (useful for threading)
	commentModel.CID = cid

	s.writeThroughWitness(collection, commentModel.RKey, cid, record)

	return commentModel, nil
}

func (s *AtprotoStore) DeleteCommentByRKey(ctx context.Context, rkey string) error {
	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return fmt.Errorf("get atp client: %w", err)
	}
	collection := s.commentCollection()
	if err := atpClient.DeleteRecord(ctx, collection, rkey); err != nil {
		return fmt.Errorf("failed to delete comment record: %w", err)
	}
	s.deleteFromWitness(collection, rkey)
	return nil
}

func (s *AtprotoStore) GetCommentsForSubject(ctx context.Context, subjectURI string) ([]*arabica.Comment, error) {
	// List all comments and filter by subject URI
	// Note: This is inefficient for large numbers of comments.
	// The firehose index provides a more efficient lookup.
	comments, err := s.ListUserComments(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*arabica.Comment
	for _, comment := range comments {
		if comment.SubjectURI == subjectURI {
			filtered = append(filtered, comment)
		}
	}

	return filtered, nil
}

func (s *AtprotoStore) ListUserComments(ctx context.Context) ([]*arabica.Comment, error) {
	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get atp client: %w", err)
	}
	records, err := atpClient.ListAllRecords(ctx, s.commentCollection())
	if err != nil {
		return nil, fmt.Errorf("failed to list comment records: %w", err)
	}

	comments := make([]*arabica.Comment, 0, len(records))

	for _, rec := range records {
		comment, err := arabica.RecordToComment(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert comment record")
			continue
		}

		// Extract rkey from URI
		if rkey := atp.RKeyFromURI(rec.URI); rkey != "" {
			comment.RKey = rkey
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func (s *AtprotoStore) Close() error {
	// No persistent connection to close for atproto
	return nil
}
