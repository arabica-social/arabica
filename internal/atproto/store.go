package atproto

import (
	"context"
	"fmt"
	"log"
	"time"

	"arabica/internal/database"
	"arabica/internal/models"

	"github.com/bluesky-social/indigo/atproto/syntax"
)

// AtprotoStore implements the database.Store interface using atproto records
type AtprotoStore struct {
	client    *Client
	did       syntax.DID
	sessionID string
}

// NewAtprotoStore creates a new atproto store for a specific user session
func NewAtprotoStore(ctx context.Context, client *Client, did syntax.DID, sessionID string) database.Store {
	return &AtprotoStore{
		client:    client,
		did:       did,
		sessionID: sessionID,
	}
}

// getContext returns a context for API calls
// Since we don't store context in the struct anymore, callers should pass context
// For now, we use background context but this should be improved
func (s *AtprotoStore) getContext() context.Context {
	return context.Background()
}

// ========== Brew Operations ==========

func (s *AtprotoStore) CreateBrew(brew *models.CreateBrewRequest, userID int) (*models.Brew, error) {
	ctx := s.getContext()

	// Build AT-URI references from rkeys
	if brew.BeanRKey == "" {
		return nil, fmt.Errorf("bean_rkey is required")
	}

	beanURI := fmt.Sprintf("at://%s/com.arabica.bean/%s", s.did.String(), brew.BeanRKey)

	var grinderURI, brewerURI string
	if brew.GrinderRKey != "" {
		grinderURI = fmt.Sprintf("at://%s/com.arabica.grinder/%s", s.did.String(), brew.GrinderRKey)
	}
	if brew.BrewerRKey != "" {
		brewerURI = fmt.Sprintf("at://%s/com.arabica.brewer/%s", s.did.String(), brew.BrewerRKey)
	}

	// Convert to models.Brew for record conversion
	brewModel := &models.Brew{
		BeanRKey:     brew.BeanRKey,
		GrinderRKey:  brew.GrinderRKey,
		BrewerRKey:   brew.BrewerRKey,
		Method:       brew.Method,
		Temperature:  brew.Temperature,
		WaterAmount:  brew.WaterAmount,
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
		Collection: "com.arabica.brew",
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

	// Fetch and resolve references to populate Bean, Grinder, Brewer
	err = ResolveBrewRefs(ctx, s.client, brewModel, beanURI, grinderURI, brewerURI, s.sessionID)
	if err != nil {
		// Non-fatal: return the brew even if we can't resolve refs
		log.Printf("Warning: failed to resolve brew references: %v", err)
	}

	return brewModel, nil
}

func (s *AtprotoStore) GetBrewByRKey(rkey string) (*models.Brew, error) {
	ctx := s.getContext()

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: "com.arabica.brew",
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brew record: %w", err)
	}

	// Build the AT-URI for this brew
	atURI := fmt.Sprintf("at://%s/com.arabica.brew/%s", s.did.String(), rkey)

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
		if _, _, beanRKey, err := ResolveATURI(beanRef); err == nil {
			brew.BeanRKey = beanRKey
		}
	}
	if grinderRef != "" {
		if _, _, grinderRKey, err := ResolveATURI(grinderRef); err == nil {
			brew.GrinderRKey = grinderRKey
		}
	}
	if brewerRef != "" {
		if _, _, brewerRKey, err := ResolveATURI(brewerRef); err == nil {
			brew.BrewerRKey = brewerRKey
		}
	}

	err = ResolveBrewRefs(ctx, s.client, brew, beanRef, grinderRef, brewerRef, s.sessionID)
	if err != nil {
		log.Printf("Warning: failed to resolve brew references: %v", err)
	}

	return brew, nil
}

func (s *AtprotoStore) ListBrews(userID int) ([]*models.Brew, error) {
	ctx := s.getContext()

	// Use ListAllRecords to handle pagination automatically
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, "com.arabica.brew")
	if err != nil {
		return nil, fmt.Errorf("failed to list brew records: %w", err)
	}

	brews := make([]*models.Brew, 0, len(output.Records))

	for _, rec := range output.Records {
		brew, err := RecordToBrew(rec.Value, rec.URI)
		if err != nil {
			log.Printf("Warning: failed to convert brew record %s: %v", rec.URI, err)
			continue
		}

		// Extract rkey from URI
		if _, _, rkey, err := ResolveATURI(rec.URI); err == nil {
			brew.RKey = rkey
		}

		// Extract and resolve references
		beanRef, _ := rec.Value["beanRef"].(string)
		grinderRef, _ := rec.Value["grinderRef"].(string)
		brewerRef, _ := rec.Value["brewerRef"].(string)

		// Extract rkeys from AT-URIs for the model
		if beanRef != "" {
			if _, _, beanRKey, err := ResolveATURI(beanRef); err == nil {
				brew.BeanRKey = beanRKey
			}
		}
		if grinderRef != "" {
			if _, _, grinderRKey, err := ResolveATURI(grinderRef); err == nil {
				brew.GrinderRKey = grinderRKey
			}
		}
		if brewerRef != "" {
			if _, _, brewerRKey, err := ResolveATURI(brewerRef); err == nil {
				brew.BrewerRKey = brewerRKey
			}
		}

		err = ResolveBrewRefs(ctx, s.client, brew, beanRef, grinderRef, brewerRef, s.sessionID)
		if err != nil {
			log.Printf("Warning: failed to resolve brew references for %s: %v", rec.URI, err)
		}

		brews = append(brews, brew)
	}

	return brews, nil
}

func (s *AtprotoStore) UpdateBrewByRKey(rkey string, brew *models.CreateBrewRequest) error {
	ctx := s.getContext()

	// Build AT-URI references from rkeys
	if brew.BeanRKey == "" {
		return fmt.Errorf("bean_rkey is required")
	}

	beanURI := fmt.Sprintf("at://%s/com.arabica.bean/%s", s.did.String(), brew.BeanRKey)

	var grinderURI, brewerURI string
	if brew.GrinderRKey != "" {
		grinderURI = fmt.Sprintf("at://%s/com.arabica.grinder/%s", s.did.String(), brew.GrinderRKey)
	}
	if brew.BrewerRKey != "" {
		brewerURI = fmt.Sprintf("at://%s/com.arabica.brewer/%s", s.did.String(), brew.BrewerRKey)
	}

	// Get the existing record to preserve createdAt
	existing, err := s.GetBrewByRKey(rkey)
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
		Collection: "com.arabica.brew",
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update brew record: %w", err)
	}

	return nil
}

func (s *AtprotoStore) DeleteBrewByRKey(rkey string) error {
	ctx := s.getContext()

	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: "com.arabica.brew",
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete brew record: %w", err)
	}

	return nil
}

// ========== Bean Operations ==========

func (s *AtprotoStore) CreateBean(bean *models.CreateBeanRequest) (*models.Bean, error) {
	ctx := s.getContext()

	var roasterURI string
	if bean.RoasterRKey != "" {
		roasterURI = fmt.Sprintf("at://%s/com.arabica.roaster/%s", s.did.String(), bean.RoasterRKey)
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
		Collection: "com.arabica.bean",
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

	return beanModel, nil
}

func (s *AtprotoStore) GetBeanByRKey(rkey string) (*models.Bean, error) {
	ctx := s.getContext()

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: "com.arabica.bean",
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get bean record: %w", err)
	}

	atURI := fmt.Sprintf("at://%s/com.arabica.bean/%s", s.did.String(), rkey)
	bean, err := RecordToBean(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert bean record: %w", err)
	}

	bean.RKey = rkey

	// Resolve roaster reference if present
	if roasterRef, ok := output.Value["roasterRef"].(string); ok && roasterRef != "" {
		// Extract rkey from roaster ref
		if _, _, roasterRKey, err := ResolveATURI(roasterRef); err == nil {
			bean.RoasterRKey = roasterRKey
		}
		// Only try to resolve if it looks like a valid AT-URI
		if len(roasterRef) > 10 && (roasterRef[:5] == "at://" || roasterRef[:4] == "did:") {
			bean.Roaster, err = ResolveRoasterRef(ctx, s.client, roasterRef, s.sessionID)
			if err != nil {
				log.Printf("Warning: failed to resolve roaster reference: %v", err)
			}
		}
	}

	return bean, nil
}

func (s *AtprotoStore) ListBeans() ([]*models.Bean, error) {
	ctx := s.getContext()

	// Use ListAllRecords to handle pagination automatically
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, "com.arabica.bean")
	if err != nil {
		return nil, fmt.Errorf("failed to list bean records: %w", err)
	}

	beans := make([]*models.Bean, 0, len(output.Records))

	for _, rec := range output.Records {
		bean, err := RecordToBean(rec.Value, rec.URI)
		if err != nil {
			log.Printf("Warning: failed to convert bean record %s: %v", rec.URI, err)
			continue
		}

		// Extract rkey from URI
		if _, _, rkey, err := ResolveATURI(rec.URI); err == nil {
			bean.RKey = rkey
		}

		// Resolve roaster reference if present
		if roasterRef, ok := rec.Value["roasterRef"].(string); ok && roasterRef != "" {
			// Extract rkey from roaster ref
			if _, _, roasterRKey, err := ResolveATURI(roasterRef); err == nil {
				bean.RoasterRKey = roasterRKey
			}
			// Only try to resolve if it looks like a valid AT-URI
			if len(roasterRef) > 10 && (roasterRef[:5] == "at://" || roasterRef[:4] == "did:") {
				bean.Roaster, err = ResolveRoasterRef(ctx, s.client, roasterRef, s.sessionID)
				if err != nil {
					log.Printf("Warning: failed to resolve roaster reference: %v", err)
				}
			}
		}

		beans = append(beans, bean)
	}

	return beans, nil
}

func (s *AtprotoStore) UpdateBeanByRKey(rkey string, bean *models.UpdateBeanRequest) error {
	ctx := s.getContext()

	// Get existing to preserve createdAt
	existing, err := s.GetBeanByRKey(rkey)
	if err != nil {
		return fmt.Errorf("failed to get existing bean: %w", err)
	}

	var roasterURI string
	if bean.RoasterRKey != "" {
		roasterURI = fmt.Sprintf("at://%s/com.arabica.roaster/%s", s.did.String(), bean.RoasterRKey)
	}

	beanModel := &models.Bean{
		Name:        bean.Name,
		Origin:      bean.Origin,
		RoastLevel:  bean.RoastLevel,
		Process:     bean.Process,
		Description: bean.Description,
		RoasterRKey: bean.RoasterRKey,
		CreatedAt:   existing.CreatedAt,
	}

	record, err := BeanToRecord(beanModel, roasterURI)
	if err != nil {
		return fmt.Errorf("failed to convert bean to record: %w", err)
	}

	err = s.client.PutRecord(ctx, s.did, s.sessionID, &PutRecordInput{
		Collection: "com.arabica.bean",
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update bean record: %w", err)
	}

	return nil
}

func (s *AtprotoStore) DeleteBeanByRKey(rkey string) error {
	ctx := s.getContext()

	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: "com.arabica.bean",
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete bean record: %w", err)
	}

	return nil
}

// ========== Roaster Operations ==========

func (s *AtprotoStore) CreateRoaster(roaster *models.CreateRoasterRequest) (*models.Roaster, error) {
	ctx := s.getContext()

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
		Collection: "com.arabica.roaster",
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

	return roasterModel, nil
}

func (s *AtprotoStore) GetRoasterByRKey(rkey string) (*models.Roaster, error) {
	ctx := s.getContext()

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: "com.arabica.roaster",
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get roaster record: %w", err)
	}

	atURI := fmt.Sprintf("at://%s/com.arabica.roaster/%s", s.did.String(), rkey)
	roaster, err := RecordToRoaster(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert roaster record: %w", err)
	}

	roaster.RKey = rkey

	return roaster, nil
}

func (s *AtprotoStore) ListRoasters() ([]*models.Roaster, error) {
	ctx := s.getContext()

	// Use ListAllRecords to handle pagination automatically
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, "com.arabica.roaster")
	if err != nil {
		return nil, fmt.Errorf("failed to list roaster records: %w", err)
	}

	roasters := make([]*models.Roaster, 0, len(output.Records))

	for _, rec := range output.Records {
		roaster, err := RecordToRoaster(rec.Value, rec.URI)
		if err != nil {
			log.Printf("Warning: failed to convert roaster record %s: %v", rec.URI, err)
			continue
		}

		// Extract rkey from URI
		if _, _, rkey, err := ResolveATURI(rec.URI); err == nil {
			roaster.RKey = rkey
		}

		roasters = append(roasters, roaster)
	}

	return roasters, nil
}

func (s *AtprotoStore) UpdateRoasterByRKey(rkey string, roaster *models.UpdateRoasterRequest) error {
	ctx := s.getContext()

	// Get existing to preserve createdAt
	existing, err := s.GetRoasterByRKey(rkey)
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
		Collection: "com.arabica.roaster",
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update roaster record: %w", err)
	}

	return nil
}

func (s *AtprotoStore) DeleteRoasterByRKey(rkey string) error {
	ctx := s.getContext()

	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: "com.arabica.roaster",
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete roaster record: %w", err)
	}

	return nil
}

// ========== Grinder Operations ==========

func (s *AtprotoStore) CreateGrinder(grinder *models.CreateGrinderRequest) (*models.Grinder, error) {
	ctx := s.getContext()

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
		Collection: "com.arabica.grinder",
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

	return grinderModel, nil
}

func (s *AtprotoStore) GetGrinderByRKey(rkey string) (*models.Grinder, error) {
	ctx := s.getContext()

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: "com.arabica.grinder",
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get grinder record: %w", err)
	}

	atURI := fmt.Sprintf("at://%s/com.arabica.grinder/%s", s.did.String(), rkey)
	grinder, err := RecordToGrinder(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert grinder record: %w", err)
	}

	grinder.RKey = rkey

	return grinder, nil
}

func (s *AtprotoStore) ListGrinders() ([]*models.Grinder, error) {
	ctx := s.getContext()

	// Use ListAllRecords to handle pagination automatically
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, "com.arabica.grinder")
	if err != nil {
		return nil, fmt.Errorf("failed to list grinder records: %w", err)
	}

	grinders := make([]*models.Grinder, 0, len(output.Records))

	for _, rec := range output.Records {
		grinder, err := RecordToGrinder(rec.Value, rec.URI)
		if err != nil {
			log.Printf("Warning: failed to convert grinder record %s: %v", rec.URI, err)
			continue
		}

		// Extract rkey from URI
		if _, _, rkey, err := ResolveATURI(rec.URI); err == nil {
			grinder.RKey = rkey
		}

		grinders = append(grinders, grinder)
	}

	return grinders, nil
}

func (s *AtprotoStore) UpdateGrinderByRKey(rkey string, grinder *models.UpdateGrinderRequest) error {
	ctx := s.getContext()

	// Get existing to preserve createdAt
	existing, err := s.GetGrinderByRKey(rkey)
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
		Collection: "com.arabica.grinder",
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update grinder record: %w", err)
	}

	return nil
}

func (s *AtprotoStore) DeleteGrinderByRKey(rkey string) error {
	ctx := s.getContext()

	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: "com.arabica.grinder",
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete grinder record: %w", err)
	}

	return nil
}

// ========== Brewer Operations ==========

func (s *AtprotoStore) CreateBrewer(brewer *models.CreateBrewerRequest) (*models.Brewer, error) {
	ctx := s.getContext()

	brewerModel := &models.Brewer{
		Name:        brewer.Name,
		Description: brewer.Description,
		CreatedAt:   time.Now(),
	}

	record, err := BrewerToRecord(brewerModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert brewer to record: %w", err)
	}

	output, err := s.client.CreateRecord(ctx, s.did, s.sessionID, &CreateRecordInput{
		Collection: "com.arabica.brewer",
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

	return brewerModel, nil
}

func (s *AtprotoStore) GetBrewerByRKey(rkey string) (*models.Brewer, error) {
	ctx := s.getContext()

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: "com.arabica.brewer",
		RKey:       rkey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get brewer record: %w", err)
	}

	atURI := fmt.Sprintf("at://%s/com.arabica.brewer/%s", s.did.String(), rkey)
	brewer, err := RecordToBrewer(output.Value, atURI)
	if err != nil {
		return nil, fmt.Errorf("failed to convert brewer record: %w", err)
	}

	brewer.RKey = rkey

	return brewer, nil
}

func (s *AtprotoStore) ListBrewers() ([]*models.Brewer, error) {
	ctx := s.getContext()

	// Use ListAllRecords to handle pagination automatically
	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, "com.arabica.brewer")
	if err != nil {
		return nil, fmt.Errorf("failed to list brewer records: %w", err)
	}

	brewers := make([]*models.Brewer, 0, len(output.Records))

	for _, rec := range output.Records {
		brewer, err := RecordToBrewer(rec.Value, rec.URI)
		if err != nil {
			log.Printf("Warning: failed to convert brewer record %s: %v", rec.URI, err)
			continue
		}

		// Extract rkey from URI
		if _, _, rkey, err := ResolveATURI(rec.URI); err == nil {
			brewer.RKey = rkey
		}

		brewers = append(brewers, brewer)
	}

	return brewers, nil
}

func (s *AtprotoStore) UpdateBrewerByRKey(rkey string, brewer *models.UpdateBrewerRequest) error {
	ctx := s.getContext()

	// Get existing to preserve createdAt
	existing, err := s.GetBrewerByRKey(rkey)
	if err != nil {
		return fmt.Errorf("failed to get existing brewer: %w", err)
	}

	brewerModel := &models.Brewer{
		Name:        brewer.Name,
		Description: brewer.Description,
		CreatedAt:   existing.CreatedAt,
	}

	record, err := BrewerToRecord(brewerModel)
	if err != nil {
		return fmt.Errorf("failed to convert brewer to record: %w", err)
	}

	err = s.client.PutRecord(ctx, s.did, s.sessionID, &PutRecordInput{
		Collection: "com.arabica.brewer",
		RKey:       rkey,
		Record:     record,
	})
	if err != nil {
		return fmt.Errorf("failed to update brewer record: %w", err)
	}

	return nil
}

func (s *AtprotoStore) DeleteBrewerByRKey(rkey string) error {
	ctx := s.getContext()

	err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: "com.arabica.brewer",
		RKey:       rkey,
	})
	if err != nil {
		return fmt.Errorf("failed to delete brewer record: %w", err)
	}

	return nil
}

func (s *AtprotoStore) Close() error {
	// No persistent connection to close for atproto
	return nil
}
