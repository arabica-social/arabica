package handlers

import (
	"context"
	"sort"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/pdewey.com/atp"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

// ProfileDataBundle holds all user data fetched from their PDS for profile display
type ProfileDataBundle struct {
	Beans      []*arabica.Bean
	Roasters   []*arabica.Roaster
	Grinders   []*arabica.Grinder
	Brewers    []*arabica.Brewer
	Brews      []*arabica.Brew
	TotalBrews int // total brew count (may differ from len(Brews) when paginated)
}

// fetchUserProfileData fetches all user data for profile display.
// Tries the witness cache first (firehose index), falling back to the PDS via publicClient.
// Brews are sorted in reverse chronological order (newest first).
func (h *Handlers) fetchUserProfileData(ctx context.Context, did string, publicClient *atp.PublicClient, brewsOffset, brewsLimit int) (*ProfileDataBundle, error) {
	// Try witness cache first — all records for this user may already be indexed
	if bundle := h.fetchProfileFromWitness(ctx, did, brewsOffset, brewsLimit); bundle != nil {
		return bundle, nil
	}

	return h.fetchProfileFromPDS(ctx, did, publicClient)
}

// fetchProfileFromWitness loads all profile data from the witness cache.
// brewsOffset and brewsLimit control pagination of the brews collection;
// other collections (beans, roasters, etc.) are always fully fetched.
// Returns nil if the witness cache is not configured or the user has no indexed records.
func (h *Handlers) fetchProfileFromWitness(ctx context.Context, did string, brewsOffset, brewsLimit int) *ProfileDataBundle {
	witnessCache := h.WitnessCache()
	if witnessCache == nil {
		return nil
	}

	// Load all collections from witness cache
	type collectionResult struct {
		collection string
		records    []*atproto.WitnessRecord
	}

	results := make(map[string][]*atproto.WitnessRecord)
	totalRecords := 0

	// Fetch non-brew collections in full (they're small)
	for _, coll := range []string{arabica.NSIDBean, arabica.NSIDRoaster, arabica.NSIDGrinder, arabica.NSIDBrewer} {
		records, err := witnessCache.ListWitnessRecords(ctx, did, coll)
		if err != nil {
			log.Debug().Err(err).Str("did", did).Str("collection", coll).Msg("witness: profile collection error")
			return nil
		}
		results[coll] = records
		totalRecords += len(records)
	}

	// Fetch brews with pagination when limit > 0
	if brewsLimit > 0 {
		records, err := witnessCache.ListWitnessRecordsPaginated(ctx, did, arabica.NSIDBrew, brewsOffset, brewsLimit)
		if err != nil {
			log.Debug().Err(err).Str("did", did).Msg("witness: profile brews paginated error")
			return nil
		}
		if records != nil {
			results[arabica.NSIDBrew] = records
			totalRecords += len(records)
		}
	} else {
		records, err := witnessCache.ListWitnessRecords(ctx, did, arabica.NSIDBrew)
		if err != nil {
			log.Debug().Err(err).Str("did", did).Msg("witness: profile brews error")
			return nil
		}
		results[arabica.NSIDBrew] = records
		totalRecords += len(records)
	}

	// If the witness cache has zero records for this user, fall back to PDS
	// (user may not have been backfilled/indexed yet)
	if totalRecords == 0 {
		return nil
	}

	metrics.WitnessCacheHitsTotal.WithLabelValues("profile").Inc()

	// Convert witness records to models
	beanMap := make(map[string]*arabica.Bean)
	beanRoasterRefMap := make(map[string]string)
	beans := make([]*arabica.Bean, 0, len(results[arabica.NSIDBean]))
	for _, wr := range results[arabica.NSIDBean] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		bean, err := arabica.RecordToBean(m, wr.URI)
		if err != nil {
			continue
		}
		bean.RKey = wr.RKey
		beans = append(beans, bean)
		beanMap[wr.URI] = bean
		if roasterRef, ok := m["roasterRef"].(string); ok && roasterRef != "" {
			beanRoasterRefMap[wr.URI] = roasterRef
			if rkey := atp.RKeyFromURI(roasterRef); rkey != "" {
				bean.RoasterRKey = rkey
			}
		}
	}

	roasterMap := make(map[string]*arabica.Roaster)
	roasters := make([]*arabica.Roaster, 0, len(results[arabica.NSIDRoaster]))
	for _, wr := range results[arabica.NSIDRoaster] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		roaster, err := arabica.RecordToRoaster(m, wr.URI)
		if err != nil {
			continue
		}
		roaster.RKey = wr.RKey
		roasters = append(roasters, roaster)
		roasterMap[wr.URI] = roaster
	}

	grinderMap := make(map[string]*arabica.Grinder)
	grinders := make([]*arabica.Grinder, 0, len(results[arabica.NSIDGrinder]))
	for _, wr := range results[arabica.NSIDGrinder] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		grinder, err := arabica.RecordToGrinder(m, wr.URI)
		if err != nil {
			continue
		}
		grinder.RKey = wr.RKey
		grinders = append(grinders, grinder)
		grinderMap[wr.URI] = grinder
	}

	brewerMap := make(map[string]*arabica.Brewer)
	brewers := make([]*arabica.Brewer, 0, len(results[arabica.NSIDBrewer]))
	for _, wr := range results[arabica.NSIDBrewer] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		brewer, err := arabica.RecordToBrewer(m, wr.URI)
		if err != nil {
			continue
		}
		brewer.RKey = wr.RKey
		brewers = append(brewers, brewer)
		brewerMap[wr.URI] = brewer
	}

	brews := make([]*arabica.Brew, 0, len(results[arabica.NSIDBrew]))
	for _, wr := range results[arabica.NSIDBrew] {
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			continue
		}
		brew, err := arabica.RecordToBrew(m, wr.URI)
		if err != nil {
			continue
		}
		brew.RKey = wr.RKey
		// Store full AT-URI refs for resolution below
		if beanRef, ok := m["beanRef"].(string); ok {
			brew.BeanRKey = beanRef
		}
		if grinderRef, ok := m["grinderRef"].(string); ok {
			brew.GrinderRKey = grinderRef
		}
		if brewerRef, ok := m["brewerRef"].(string); ok {
			brew.BrewerRKey = brewerRef
		}
		brews = append(brews, brew)
	}

	// Resolve references (same logic as PDS path)
	for _, bean := range beans {
		if roasterRef, found := beanRoasterRefMap[atp.BuildATURI(did, arabica.NSIDBean, bean.RKey)]; found {
			if roaster, found := roasterMap[roasterRef]; found {
				bean.Roaster = roaster
			}
		}
	}

	for _, brew := range brews {
		if brew.BeanRKey != "" {
			if bean, found := beanMap[brew.BeanRKey]; found {
				brew.Bean = bean
			}
		}
		if brew.GrinderRKey != "" {
			if grinder, found := grinderMap[brew.GrinderRKey]; found {
				brew.GrinderObj = grinder
			}
		}
		if brew.BrewerRKey != "" {
			if brewer, found := brewerMap[brew.BrewerRKey]; found {
				brew.BrewerObj = brewer
			}
		}
	}

	sort.Slice(brews, func(i, j int) bool {
		return brews[i].CreatedAt.After(brews[j].CreatedAt)
	})

	// Get total brew count from witness cache for accurate stats display.
	totalBrews := len(brews)
	if brewsLimit > 0 {
		if c, err := witnessCache.CountWitnessRecords(ctx, did, arabica.NSIDBrew); err == nil {
			totalBrews = c
		}
	}

	return &ProfileDataBundle{
		Beans:      beans,
		Roasters:   roasters,
		Grinders:   grinders,
		Brewers:    brewers,
		Brews:      brews,
		TotalBrews: totalBrews,
	}
}

// fetchProfileFromPDS fetches all user data from their PDS via publicClient in parallel.
func (h *Handlers) fetchProfileFromPDS(ctx context.Context, did string, publicClient *atp.PublicClient) (*ProfileDataBundle, error) {
	metrics.WitnessCacheMissesTotal.WithLabelValues("profile").Inc()

	// Fetch all user data in parallel
	g, gCtx := errgroup.WithContext(ctx)

	var brews []*arabica.Brew
	var beans []*arabica.Bean
	var roasters []*arabica.Roaster
	var grinders []*arabica.Grinder
	var brewers []*arabica.Brewer

	// Maps for resolving references
	var beanMap map[string]*arabica.Bean
	var beanRoasterRefMap map[string]string
	var roasterMap map[string]*arabica.Roaster
	var brewerMap map[string]*arabica.Brewer
	var grinderMap map[string]*arabica.Grinder

	// Fetch beans
	g.Go(func() error {
		records, _, err := publicClient.ListPublicRecords(gCtx, did, arabica.NSIDBean, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			return err
		}
		beanMap = make(map[string]*arabica.Bean)
		beanRoasterRefMap = make(map[string]string)
		beans = make([]*arabica.Bean, 0, len(records))
		for _, record := range records {
			bean, err := arabica.RecordToBean(record.Value, record.URI)
			if err != nil {
				continue
			}
			beans = append(beans, bean)
			beanMap[record.URI] = bean
			if roasterRef, ok := record.Value["roasterRef"].(string); ok && roasterRef != "" {
				beanRoasterRefMap[record.URI] = roasterRef
			}
		}
		return nil
	})

	// Fetch roasters
	g.Go(func() error {
		records, _, err := publicClient.ListPublicRecords(gCtx, did, arabica.NSIDRoaster, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			return err
		}
		roasterMap = make(map[string]*arabica.Roaster)
		roasters = make([]*arabica.Roaster, 0, len(records))
		for _, record := range records {
			roaster, err := arabica.RecordToRoaster(record.Value, record.URI)
			if err != nil {
				continue
			}
			roasters = append(roasters, roaster)
			roasterMap[record.URI] = roaster
		}
		return nil
	})

	// Fetch grinders
	g.Go(func() error {
		records, _, err := publicClient.ListPublicRecords(gCtx, did, arabica.NSIDGrinder, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			return err
		}
		grinderMap = make(map[string]*arabica.Grinder)
		grinders = make([]*arabica.Grinder, 0, len(records))
		for _, record := range records {
			grinder, err := arabica.RecordToGrinder(record.Value, record.URI)
			if err != nil {
				continue
			}
			grinders = append(grinders, grinder)
			grinderMap[record.URI] = grinder
		}
		return nil
	})

	// Fetch brewers
	g.Go(func() error {
		records, _, err := publicClient.ListPublicRecords(gCtx, did, arabica.NSIDBrewer, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			return err
		}
		brewerMap = make(map[string]*arabica.Brewer)
		brewers = make([]*arabica.Brewer, 0, len(records))
		for _, record := range records {
			brewer, err := arabica.RecordToBrewer(record.Value, record.URI)
			if err != nil {
				continue
			}
			brewers = append(brewers, brewer)
			brewerMap[record.URI] = brewer
		}
		return nil
	})

	// Fetch brews
	g.Go(func() error {
		records, _, err := publicClient.ListPublicRecords(gCtx, did, arabica.NSIDBrew, atp.ListPublicRecordsOpts{Limit: 100, Reverse: true})
		if err != nil {
			return err
		}
		brews = make([]*arabica.Brew, 0, len(records))
		for _, record := range records {
			brew, err := arabica.RecordToBrew(record.Value, record.URI)
			if err != nil {
				continue
			}
			// Store the raw record for reference resolution later
			brew.BeanRKey = ""
			if beanRef, ok := record.Value["beanRef"].(string); ok {
				brew.BeanRKey = beanRef
			}
			if grinderRef, ok := record.Value["grinderRef"].(string); ok {
				brew.GrinderRKey = grinderRef
			}
			if brewerRef, ok := record.Value["brewerRef"].(string); ok {
				brew.BrewerRKey = brewerRef
			}
			brews = append(brews, brew)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Resolve references for beans (roaster refs)
	for _, bean := range beans {
		if roasterRef, found := beanRoasterRefMap[atp.BuildATURI(did, arabica.NSIDBean, bean.RKey)]; found {
			if roaster, found := roasterMap[roasterRef]; found {
				bean.Roaster = roaster
			}
		}
	}

	// Resolve references for brews
	for _, brew := range brews {
		// Resolve bean reference
		if brew.BeanRKey != "" {
			if bean, found := beanMap[brew.BeanRKey]; found {
				brew.Bean = bean
			}
		}
		// Resolve grinder reference
		if brew.GrinderRKey != "" {
			if grinder, found := grinderMap[brew.GrinderRKey]; found {
				brew.GrinderObj = grinder
			}
		}
		// Resolve brewer reference
		if brew.BrewerRKey != "" {
			if brewer, found := brewerMap[brew.BrewerRKey]; found {
				brew.BrewerObj = brewer
			}
		}
	}

	// Sort brews in reverse chronological order (newest first)
	sort.Slice(brews, func(i, j int) bool {
		return brews[i].CreatedAt.After(brews[j].CreatedAt)
	})

	return &ProfileDataBundle{
		Beans:      beans,
		Roasters:   roasters,
		Grinders:   grinders,
		Brewers:    brewers,
		Brews:      brews,
		TotalBrews: len(brews),
	}, nil
}
