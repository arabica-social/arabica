package arabicastore

import (
	"context"
	"fmt"
	"time"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/pdewey.com/atp"

	"github.com/rs/zerolog/log"
)

// AtprotoStore adapts the generic atproto store to Arabica's typed Store interface.
type AtprotoStore struct {
	*atproto.AtprotoStore
}

var _ Store = (*AtprotoStore)(nil)

var (
	roasterCodec = arabica.AtprotoRoasterCodec
	grinderCodec = arabica.AtprotoGrinderCodec
	brewerCodec  = arabica.AtprotoBrewerCodec
	beanCodec    = arabica.AtprotoBeanCodec
	recipeCodec  = arabica.AtprotoRecipeCodec
)

// NewAtprotoStore wraps a generic AT Protocol store with Arabica typed methods.
func NewAtprotoStore(core *atproto.AtprotoStore) *AtprotoStore {
	return &AtprotoStore{AtprotoStore: core}
}

// RawAtprotoStore exposes the generic store for shared same-owner view helpers.
func (s *AtprotoStore) RawAtprotoStore() *atproto.AtprotoStore {
	return s.AtprotoStore
}

func (s *AtprotoStore) resolveBrewRefsFromWitness(ctx context.Context, brew *arabica.Brew, record map[string]any) {
	lookup := func(refURI string) (map[string]any, bool) {
		wr := s.AtprotoStore.WitnessRecordByURI(ctx, refURI)
		if wr == nil {
			return nil, false
		}
		record, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			return nil, false
		}
		return record, true
	}
	arabica.HydrateBrewRefs(brew, record, lookup)
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
	atpClient, err := s.AtprotoStore.ATPClient(ctx)
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
	beanURI := atp.BuildATURI(s.DID(), arabica.NSIDBean, brew.BeanRKey)
	var grinderURI, brewerURI, recipeURI string
	if brew.GrinderRKey != "" {
		grinderURI = atp.BuildATURI(s.DID(), arabica.NSIDGrinder, brew.GrinderRKey)
	}
	if brew.BrewerRKey != "" {
		brewerURI = atp.BuildATURI(s.DID(), arabica.NSIDBrewer, brew.BrewerRKey)
	}
	if brew.RecipeRKey != "" {
		recipeOwner := s.DID()
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
	rkey, _, err := s.AtprotoStore.PutRecord(ctx, arabica.NSIDBrew, "", record)
	if err != nil {
		return nil, err
	}
	model.RKey = rkey
	// Populate Bean/GrinderObj/BrewerObj for the response.
	if atpClient, err := s.AtprotoStore.ATPClient(ctx); err == nil {
		if err := arabica.ResolveBrewRefs(ctx, atpClient, model, beanURI, grinderURI, brewerURI); err != nil {
			log.Warn().Err(err).Str("brew_rkey", rkey).Msg("resolve brew refs after create")
		}
	}
	return model, nil
}

func (s *AtprotoStore) GetBrewByRKey(ctx context.Context, rkey string) (*arabica.Brew, error) {
	rec, uri, _, hit, fromWitness, err := s.AtprotoStore.FetchRecordSource(ctx, arabica.NSIDBrew, rkey)
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

// GetBrewRecordByRKey fetches a brew by rkey and returns it with its AT
// Protocol metadata. Brew's ref-resolution is too complex for the
// EntityCodec PostGet hook (bulk witness-batch lookups), so this stays
// bespoke even though the wrapper shape is shared.
func (s *AtprotoStore) GetBrewRecordByRKey(ctx context.Context, rkey string) (*atproto.EntityRecord[arabica.Brew], error) {
	rec, uri, cid, hit, fromWitness, err := s.AtprotoStore.FetchRecordSource(ctx, arabica.NSIDBrew, rkey)
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
	return &atproto.EntityRecord[arabica.Brew]{Model: brew, URI: uri, CID: cid}, nil
}

func (s *AtprotoStore) ListBrews(ctx context.Context, userID int, offset, limit int) ([]*arabica.Brew, error) {
	// For paginated requests, skip session cache and go directly to the witness
	// cache (local SQLite) with LIMIT/OFFSET. The session cache is only useful for
	// the "fetch all" case (e.g., export, client-side cache).
	if limit > 0 {
		return s.listBrewsPage(ctx, offset, limit)
	}

	// Non-paginated: use session cache then fetch all records.
	if uc := s.AtprotoStore.Cache().Get(s.AtprotoStore.SessionID()); uc != nil && atproto.CachedSlice[arabica.Brew](uc, arabica.NSIDBrew) != nil && uc.IsValid() {
		return atproto.CachedSlice[arabica.Brew](uc, arabica.NSIDBrew), nil
	}
	raws, err := s.AtprotoStore.FetchAllRecords(ctx, arabica.NSIDBrew)
	if err != nil {
		return nil, err
	}
	brews := s.convertBrewRecords(raws)
	// Resolve references in bulk
	s.resolveBrewReferences(ctx, brews)
	s.AtprotoStore.Cache().SetRecords(s.AtprotoStore.SessionID(), arabica.NSIDBrew, brews)
	s.AtprotoStore.Cache().ClearDirty(s.AtprotoStore.SessionID(), arabica.NSIDBrew)
	return brews, nil
}

// listBrewsPage fetches a single page of brews from the witness cache with
// LIMIT/OFFSET, resolving references in bulk.
func (s *AtprotoStore) listBrewsPage(ctx context.Context, offset, limit int) ([]*arabica.Brew, error) {
	raws, err := s.AtprotoStore.FetchPaginatedRecords(ctx, arabica.NSIDBrew, offset, limit)
	if err != nil {
		return nil, err
	}
	brews := s.convertBrewRecords(raws)
	s.resolveBrewReferences(ctx, brews)
	return brews, nil
}

// convertBrewRecords converts raw records to typed Brew models.
func (s *AtprotoStore) convertBrewRecords(raws []atproto.RawRecord) []*arabica.Brew {
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
	grinders, _ := s.listGrinders(ctx)
	brewers, _ := s.listBrewers(ctx)
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
	beanURI := atp.BuildATURI(s.DID(), arabica.NSIDBean, brew.BeanRKey)
	var grinderURI, brewerURI, recipeURI string
	if brew.GrinderRKey != "" {
		grinderURI = atp.BuildATURI(s.DID(), arabica.NSIDGrinder, brew.GrinderRKey)
	}
	if brew.BrewerRKey != "" {
		brewerURI = atp.BuildATURI(s.DID(), arabica.NSIDBrewer, brew.BrewerRKey)
	}
	if brew.RecipeRKey != "" {
		recipeOwner := s.DID()
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
	_, _, err = s.AtprotoStore.PutRecord(ctx, arabica.NSIDBrew, rkey, record)
	return err
}

func (s *AtprotoStore) DeleteBrewByRKey(ctx context.Context, rkey string) error {
	return s.AtprotoStore.RemoveRecord(ctx, arabica.NSIDBrew, rkey)
}

func (s *AtprotoStore) GetBeanRecordByRKey(ctx context.Context, rkey string) (*atproto.EntityRecord[arabica.Bean], error) {
	return atproto.GetEntityRecord(ctx, s, beanCodec, rkey)
}

func (s *AtprotoStore) GetRoasterRecordByRKey(ctx context.Context, rkey string) (*atproto.EntityRecord[arabica.Roaster], error) {
	return atproto.GetEntityRecord(ctx, s, roasterCodec, rkey)
}

func (s *AtprotoStore) GetGrinderRecordByRKey(ctx context.Context, rkey string) (*atproto.EntityRecord[arabica.Grinder], error) {
	return atproto.GetEntityRecord(ctx, s, grinderCodec, rkey)
}

func (s *AtprotoStore) GetBrewerRecordByRKey(ctx context.Context, rkey string) (*atproto.EntityRecord[arabica.Brewer], error) {
	return atproto.GetEntityRecord(ctx, s, brewerCodec, rkey)
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
	if wr := s.AtprotoStore.WitnessRecordByURI(ctx, roasterRef); wr != nil {
		if m, err := atproto.WitnessRecordToMap(wr); err == nil {
			if roaster, err := arabica.RecordToRoaster(m, wr.URI); err == nil {
				roaster.RKey = wr.RKey
				bean.Roaster = roaster
				return
			}
		}
	}
	// Fall back to PDS resolver
	if len(roasterRef) > 10 && (roasterRef[:5] == "at://" || roasterRef[:4] == "did:") {
		atpClient, err := s.AtprotoStore.ATPClient(ctx)
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

func (s *AtprotoStore) ResolveBeanRefs(ctx context.Context, bean *arabica.Bean, record map[string]any) {
	s.resolveBeanRefs(ctx, bean, record)
}

func (s *AtprotoStore) CreateBean(ctx context.Context, bean *arabica.CreateBeanRequest) (*arabica.Bean, error) {
	return atproto.CreateEntity(ctx, s, beanCodec, &arabica.Bean{
		Name:        bean.Name,
		Origin:      bean.Origin,
		Variety:     bean.Variety,
		RoastLevel:  bean.RoastLevel,
		Process:     bean.Process,
		Description: bean.Description,
		Link:        bean.Link,
		RoasterRKey: bean.RoasterRKey,
		Rating:      bean.Rating,
		SourceRef:   bean.SourceRef,
		CreatedAt:   time.Now().UTC(),
	})
}

func (s *AtprotoStore) GetBeanByRKey(ctx context.Context, rkey string) (*arabica.Bean, error) {
	return atproto.GetEntity(ctx, s, beanCodec, rkey)
}

func (s *AtprotoStore) ListBeans(ctx context.Context) ([]*arabica.Bean, error) {
	return atproto.ListEntity(ctx, s, beanCodec, func() []*arabica.Bean {
		return atproto.CachedSlice[arabica.Bean](s.AtprotoStore.Cache().Get(s.AtprotoStore.SessionID()), arabica.NSIDBean)
	})
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
	return atproto.UpdateEntity(ctx, s, beanCodec, rkey, &arabica.Bean{
		Name:        bean.Name,
		Origin:      bean.Origin,
		Variety:     bean.Variety,
		RoastLevel:  bean.RoastLevel,
		Process:     bean.Process,
		Description: bean.Description,
		Link:        bean.Link,
		RoasterRKey: bean.RoasterRKey,
		Rating:      bean.Rating,
		Closed:      bean.Closed,
		SourceRef:   bean.SourceRef,
		CreatedAt:   existing.CreatedAt,
	})
}

func (s *AtprotoStore) DeleteBeanByRKey(ctx context.Context, rkey string) error {
	return atproto.DeleteEntity(ctx, s, arabica.NSIDBean, rkey)
}

// ========== Roaster Operations ==========

func (s *AtprotoStore) CreateRoaster(ctx context.Context, roaster *arabica.CreateRoasterRequest) (*arabica.Roaster, error) {
	return atproto.CreateEntity(ctx, s, roasterCodec, &arabica.Roaster{
		Name:      roaster.Name,
		Location:  roaster.Location,
		Website:   roaster.Website,
		SourceRef: roaster.SourceRef,
		CreatedAt: time.Now().UTC(),
	})
}

func (s *AtprotoStore) GetRoasterByRKey(ctx context.Context, rkey string) (*arabica.Roaster, error) {
	return atproto.GetEntity(ctx, s, roasterCodec, rkey)
}

func (s *AtprotoStore) ListRoasters(ctx context.Context) ([]*arabica.Roaster, error) {
	return atproto.ListEntity(ctx, s, roasterCodec, func() []*arabica.Roaster {
		return atproto.CachedSlice[arabica.Roaster](s.AtprotoStore.Cache().Get(s.AtprotoStore.SessionID()), arabica.NSIDRoaster)
	})
}

func (s *AtprotoStore) UpdateRoasterByRKey(ctx context.Context, rkey string, roaster *arabica.UpdateRoasterRequest) error {
	existing, err := s.GetRoasterByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("get existing roaster: %w", err)
	}
	err = atproto.UpdateEntity(ctx, s, roasterCodec, rkey, &arabica.Roaster{
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
	s.AtprotoStore.Cache().InvalidateRecords(s.AtprotoStore.SessionID(), arabica.NSIDBean)
	return nil
}

func (s *AtprotoStore) DeleteRoasterByRKey(ctx context.Context, rkey string) error {
	if err := atproto.DeleteEntity(ctx, s, arabica.NSIDRoaster, rkey); err != nil {
		return err
	}
	// Beans denormalize roaster data; invalidate them too.
	s.AtprotoStore.Cache().InvalidateRecords(s.AtprotoStore.SessionID(), arabica.NSIDBean)
	return nil
}

// ========== Grinder/Brewer compatibility helpers ==========

func (s *AtprotoStore) CreateGrinder(ctx context.Context, grinder *arabica.CreateGrinderRequest) (*arabica.Grinder, error) {
	return atproto.CreateEntity(ctx, s, grinderCodec, &arabica.Grinder{
		Name:        grinder.Name,
		GrinderType: grinder.GrinderType,
		BurrType:    grinder.BurrType,
		Notes:       grinder.Notes,
		Link:        grinder.Link,
		SourceRef:   grinder.SourceRef,
		CreatedAt:   time.Now().UTC(),
	})
}

func (s *AtprotoStore) GetGrinderByRKey(ctx context.Context, rkey string) (*arabica.Grinder, error) {
	return atproto.GetEntity(ctx, s, grinderCodec, rkey)
}

func (s *AtprotoStore) CreateBrewer(ctx context.Context, brewer *arabica.CreateBrewerRequest) (*arabica.Brewer, error) {
	return atproto.CreateEntity(ctx, s, brewerCodec, &arabica.Brewer{
		Name:        brewer.Name,
		BrewerType:  brewer.BrewerType,
		Description: brewer.Description,
		Link:        brewer.Link,
		SourceRef:   brewer.SourceRef,
		CreatedAt:   time.Now().UTC(),
	})
}

func (s *AtprotoStore) GetBrewerByRKey(ctx context.Context, rkey string) (*arabica.Brewer, error) {
	return atproto.GetEntity(ctx, s, brewerCodec, rkey)
}

// ========== Grinder/Brewer list helpers ==========

func (s *AtprotoStore) listGrinders(ctx context.Context) ([]*arabica.Grinder, error) {
	return atproto.ListEntity(ctx, s, grinderCodec, func() []*arabica.Grinder {
		return atproto.CachedSlice[arabica.Grinder](s.AtprotoStore.Cache().Get(s.AtprotoStore.SessionID()), arabica.NSIDGrinder)
	})
}

func (s *AtprotoStore) listBrewers(ctx context.Context) ([]*arabica.Brewer, error) {
	return atproto.ListEntity(ctx, s, brewerCodec, func() []*arabica.Brewer {
		return atproto.CachedSlice[arabica.Brewer](s.AtprotoStore.Cache().Get(s.AtprotoStore.SessionID()), arabica.NSIDBrewer)
	})
}

// ========== Recipe Operations ==========

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
	if wr := s.AtprotoStore.WitnessRecordByURI(ctx, brewerRef); wr != nil {
		if m, err := atproto.WitnessRecordToMap(wr); err == nil {
			if brewer, err := arabica.RecordToBrewer(m, wr.URI); err == nil {
				brewer.RKey = wr.RKey
				recipe.BrewerObj = brewer
				return
			}
		}
	}
	atpClient, err := s.AtprotoStore.ATPClient(ctx)
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

func (s *AtprotoStore) ResolveRecipeRefs(ctx context.Context, recipe *arabica.Recipe, record map[string]any) {
	s.resolveRecipeRefs(ctx, recipe, record)
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
	return atproto.CreateEntity(ctx, s, recipeCodec, recipeModelFromCreate(req))
}

func (s *AtprotoStore) GetRecipeByRKey(ctx context.Context, rkey string) (*arabica.Recipe, error) {
	return atproto.GetEntity(ctx, s, recipeCodec, rkey)
}

func (s *AtprotoStore) GetRecipeRecordByRKey(ctx context.Context, rkey string) (*atproto.EntityRecord[arabica.Recipe], error) {
	return atproto.GetEntityRecord(ctx, s, recipeCodec, rkey)
}

func (s *AtprotoStore) ListRecipes(ctx context.Context) ([]*arabica.Recipe, error) {
	return atproto.ListEntity(ctx, s, recipeCodec, func() []*arabica.Recipe {
		return atproto.CachedSlice[arabica.Recipe](s.AtprotoStore.Cache().Get(s.AtprotoStore.SessionID()), arabica.NSIDRecipe)
	})
}

func (s *AtprotoStore) UpdateRecipeByRKey(ctx context.Context, rkey string, req *arabica.UpdateRecipeRequest) error {
	existing, err := s.GetRecipeByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("get existing recipe: %w", err)
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
	return atproto.UpdateEntity(ctx, s, recipeCodec, rkey, model)
}

func (s *AtprotoStore) DeleteRecipeByRKey(ctx context.Context, rkey string) error {
	return atproto.DeleteEntity(ctx, s, arabica.NSIDRecipe, rkey)
}
