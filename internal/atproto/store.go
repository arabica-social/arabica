package atproto

import (
	"context"
	"fmt"
	"time"

	"arabica/internal/database"
	"arabica/internal/models"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
)

// AtprotoStore implements the database.Store interface using atproto records.
// Context is passed as a parameter to each method rather than stored in the struct,
// following Go best practices for context propagation.
type AtprotoStore struct {
	client    *Client
	did       syntax.DID
	sessionID string
	cache     *SessionCache
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

// ========== Brew Operations ==========

func (s *AtprotoStore) CreateBrew(ctx context.Context, brew *models.CreateBrewRequest, userID int) (*models.Brew, error) {
	// Build AT-URI references from rkeys
	if brew.BeanRKey == "" {
		return nil, fmt.Errorf("bean_rkey is required")
	}

	beanURI := BuildATURI(s.did.String(), NSIDBean, brew.BeanRKey)

	var grinderURI, brewerURI string
	if brew.GrinderRKey != "" {
		grinderURI = BuildATURI(s.did.String(), NSIDGrinder, brew.GrinderRKey)
	}
	if brew.BrewerRKey != "" {
		brewerURI = BuildATURI(s.did.String(), NSIDBrewer, brew.BrewerRKey)
	}

	// Convert to models.Brew for record conversion
	brewModel := &models.Brew{
		BeanRKey:     brew.BeanRKey,
		GrinderRKey:  brew.GrinderRKey,
		BrewerRKey:   brew.BrewerRKey,
		Method:       brew.Method,
		Temperature:  brew.Temperature,
		WaterAmount:  brew.WaterAmount,
		CoffeeAmount: brew.CoffeeAmount,
		TimeSeconds:  brew.TimeSeconds,
		GrindSize:    brew.GrindSize,
		TastingNotes: brew.TastingNotes,
		Rating:       brew.Rating,
		CreatedAt:    time.Now(),
	}

	// Convert pours
	if len(brew.Pours) > 0 {
		brewModel.Pours = make([]*models.Pour, len(brew.Pours))
		for i, pour := range brew.Pours {
			brewModel.Pours[i] = &models.Pour{
				WaterAmount: pour.WaterAmount,
				TimeSeconds: pour.TimeSeconds,
			}
		}
	}

	// Convert to atproto record
	record, err := BrewToRecord(brewModel, beanURI, grinderURI, brewerURI)
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

	// Invalidate brews cache
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
	beanRef, _ := output.Value["beanRef"].(string)
	grinderRef, _ := output.Value["grinderRef"].(string)
	brewerRef, _ := output.Value["brewerRef"].(string)

	// Extract rkeys from AT-URIs for the model
	if beanRef != "" {
		if components, err := ResolveATURI(beanRef); err == nil {
			brew.BeanRKey = components.RKey
		}
	}
	if grinderRef != "" {
		if components, err := ResolveATURI(grinderRef); err == nil {
			brew.GrinderRKey = components.RKey
		}
	}
	if brewerRef != "" {
		if components, err := ResolveATURI(brewerRef); err == nil {
			brew.BrewerRKey = components.RKey
		}
	}

	err = ResolveBrewRefs(ctx, s.client, brew, beanRef, grinderRef, brewerRef, s.sessionID)
	if err != nil {
		log.Warn().Err(err).Str("brew_rkey", rkey).Msg("Failed to resolve brew references")
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
	beanRef, _ := output.Value["beanRef"].(string)
	grinderRef, _ := output.Value["grinderRef"].(string)
	brewerRef, _ := output.Value["brewerRef"].(string)

	// Extract rkeys from AT-URIs for the model
	if beanRef != "" {
		if components, err := ResolveATURI(beanRef); err == nil {
			brew.BeanRKey = components.RKey
		}
	}
	if grinderRef != "" {
		if components, err := ResolveATURI(grinderRef); err == nil {
			brew.GrinderRKey = components.RKey
		}
	}
	if brewerRef != "" {
		if components, err := ResolveATURI(brewerRef); err == nil {
			brew.BrewerRKey = components.RKey
		}
	}

	err = ResolveBrewRefs(ctx, s.client, brew, beanRef, grinderRef, brewerRef, s.sessionID)
	if err != nil {
		log.Warn().Err(err).Str("brew_rkey", rkey).Msg("Failed to resolve brew references")
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
	if userCache != nil && userCache.Brews != nil && userCache.IsValid() {
		return userCache.Brews, nil
	}

	// Use ListAllRecords to handle pagination automatically
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, NSIDBrew)
	if err != nil {
		return nil, fmt.Errorf("failed to list brew records: %w", err)
	}

	brews := make([]*models.Brew, 0, len(output.Records))

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
		beanRef, _ := rec.Value["beanRef"].(string)
		grinderRef, _ := rec.Value["grinderRef"].(string)
		brewerRef, _ := rec.Value["brewerRef"].(string)

		if beanRef != "" {
			if components, err := ResolveATURI(beanRef); err == nil {
				brew.BeanRKey = components.RKey
			}
		}
		if grinderRef != "" {
			if components, err := ResolveATURI(grinderRef); err == nil {
				brew.GrinderRKey = components.RKey
			}
		}
		if brewerRef != "" {
			if components, err := ResolveATURI(brewerRef); err == nil {
				brew.BrewerRKey = components.RKey
			}
		}

		brews = append(brews, brew)
	}

	// Resolve references using cached data instead of N+1 queries
	// This fetches beans/grinders/brewers once (from cache if available)
	// then links them to brews in memory
	beans, _ := s.ListBeans(ctx)
	grinders, _ := s.ListGrinders(ctx)
	brewers, _ := s.ListBrewers(ctx)
	roasters, _ := s.ListRoasters(ctx)

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
	}

	// Update cache
	s.cache.SetBrews(s.sessionID, brews)

	return brews, nil
}

func (s *AtprotoStore) UpdateBrewByRKey(ctx context.Context, rkey string, brew *models.CreateBrewRequest) error {
	// Build AT-URI references from rkeys
	if brew.BeanRKey == "" {
		return fmt.Errorf("bean_rkey is required")
	}

	beanURI := BuildATURI(s.did.String(), NSIDBean, brew.BeanRKey)

	var grinderURI, brewerURI string
	if brew.GrinderRKey != "" {
		grinderURI = BuildATURI(s.did.String(), NSIDGrinder, brew.GrinderRKey)
	}
	if brew.BrewerRKey != "" {
		brewerURI = BuildATURI(s.did.String(), NSIDBrewer, brew.BrewerRKey)
	}

	// Get the existing record to preserve createdAt
	existing, err := s.GetBrewByRKey(ctx, rkey)
	if err != nil {
		return fmt.Errorf("failed to get existing brew: %w", err)
	}

	// Convert to models.Brew
	brewModel := &models.Brew{
		BeanRKey:     brew.BeanRKey,
		GrinderRKey:  brew.GrinderRKey,
		BrewerRKey:   brew.BrewerRKey,
		Method:       brew.Method,
		Temperature:  brew.Temperature,
		WaterAmount:  brew.WaterAmount,
		CoffeeAmount: brew.CoffeeAmount,
		TimeSeconds:  brew.TimeSeconds,
		GrindSize:    brew.GrindSize,
		TastingNotes: brew.TastingNotes,
		Rating:       brew.Rating,
		CreatedAt:    existing.CreatedAt, // Preserve original creation time
	}

	// Convert pours
	if len(brew.Pours) > 0 {
		brewModel.Pours = make([]*models.Pour, len(brew.Pours))
		for i, pour := range brew.Pours {
			brewModel.Pours[i] = &models.Pour{
				WaterAmount: pour.WaterAmount,
				TimeSeconds: pour.TimeSeconds,
			}
		}
	}

	// Convert to atproto record
	record, err := BrewToRecord(brewModel, beanURI, grinderURI, brewerURI)
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

	// Invalidate brews cache
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

	// Invalidate brews cache
	s.cache.InvalidateBrews(s.sessionID)

	return nil
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
		RoastLevel:  bean.RoastLevel,
		Process:     bean.Process,
		Description: bean.Description,
		RoasterRKey: bean.RoasterRKey,
		CreatedAt:   time.Now(),
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

	// Invalidate cache
	s.cache.InvalidateBeans(s.sessionID)

	return beanModel, nil
}

func (s *AtprotoStore) GetBeanByRKey(ctx context.Context, rkey string) (*models.Bean, error) {
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
	if userCache != nil && userCache.Beans != nil && userCache.IsValid() {
		return userCache.Beans, nil
	}

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

	// Update cache
	s.cache.SetBeans(s.sessionID, beans)

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
		RoastLevel:  bean.RoastLevel,
		Process:     bean.Process,
		Description: bean.Description,
		RoasterRKey: bean.RoasterRKey,
		Closed:      bean.Closed,
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

	// Invalidate cache
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

	// Invalidate cache
	s.cache.InvalidateBeans(s.sessionID)

	return nil
}

// ========== Roaster Operations ==========

func (s *AtprotoStore) CreateRoaster(ctx context.Context, roaster *models.CreateRoasterRequest) (*models.Roaster, error) {
	roasterModel := &models.Roaster{
		Name:      roaster.Name,
		Location:  roaster.Location,
		Website:   roaster.Website,
		CreatedAt: time.Now(),
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

	// Invalidate cache
	s.cache.InvalidateRoasters(s.sessionID)

	return roasterModel, nil
}

func (s *AtprotoStore) GetRoasterByRKey(ctx context.Context, rkey string) (*models.Roaster, error) {
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
	if userCache != nil && userCache.Roasters != nil && userCache.IsValid() {
		return userCache.Roasters, nil
	}

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

	// Update cache
	s.cache.SetRoasters(s.sessionID, roasters)

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

	// Invalidate cache
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

	// Invalidate cache
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
		CreatedAt:   time.Now(),
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

	// Invalidate cache
	s.cache.InvalidateGrinders(s.sessionID)

	return grinderModel, nil
}

func (s *AtprotoStore) GetGrinderByRKey(ctx context.Context, rkey string) (*models.Grinder, error) {
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
	if userCache != nil && userCache.Grinders != nil && userCache.IsValid() {
		return userCache.Grinders, nil
	}

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

	// Update cache
	s.cache.SetGrinders(s.sessionID, grinders)

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

	// Invalidate cache
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

	// Invalidate cache
	s.cache.InvalidateGrinders(s.sessionID)

	return nil
}

// ========== Brewer Operations ==========

func (s *AtprotoStore) CreateBrewer(ctx context.Context, brewer *models.CreateBrewerRequest) (*models.Brewer, error) {
	brewerModel := &models.Brewer{
		Name:        brewer.Name,
		BrewerType:  brewer.BrewerType,
		Description: brewer.Description,
		CreatedAt:   time.Now(),
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

	// Invalidate cache
	s.cache.InvalidateBrewers(s.sessionID)

	return brewerModel, nil
}

func (s *AtprotoStore) GetBrewerByRKey(ctx context.Context, rkey string) (*models.Brewer, error) {
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
	if userCache != nil && userCache.Brewers != nil && userCache.IsValid() {
		return userCache.Brewers, nil
	}

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

	// Update cache
	s.cache.SetBrewers(s.sessionID, brewers)

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

	// Invalidate cache
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

	// Invalidate cache
	s.cache.InvalidateBrewers(s.sessionID)

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
		CreatedAt:  time.Now(),
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

func (s *AtprotoStore) Close() error {
	// No persistent connection to close for atproto
	return nil
}
