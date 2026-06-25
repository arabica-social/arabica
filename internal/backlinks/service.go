// Package backlinks computes lightweight community backlinks for indexed
// records: copies/forks via sourceRef, fuzzy same-entity matches, and typed
// usage references such as brews that reference a bean.
package backlinks

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"sync"
	"time"

	"tangled.org/pdewey.com/atp"
)

type UsageRef struct {
	Collection string
	Field      string
	Label      string
	Key        string
}

func (r UsageRef) key() string {
	if r.Key != "" {
		return r.Key
	}
	if r.Label != "" {
		return r.Label
	}
	return r.Field
}

type EntityConfig struct {
	Collection string
	Noun       string
	AllFields  []string
	DedupKey   func(fields map[string]string) string
	UsageRefs  []UsageRef
}

var (
	registryMu sync.RWMutex
	registry   = map[string]EntityConfig{}
)

func Register(collection string, cfg EntityConfig) {
	cfg.Collection = collection
	registryMu.Lock()
	registry[collection] = cfg
	registryMu.Unlock()
}

func ConfigFor(collection string) (EntityConfig, bool) {
	registryMu.RLock()
	cfg, ok := registry[collection]
	registryMu.RUnlock()
	return cfg, ok
}

const (
	maxChainDepth   = 5
	maxChainRecords = 500
	maxEntries      = 200
)

type Entry struct {
	DID         string
	Handle      string
	DisplayName string
	AvatarURL   string
	RecordURI   string
	Collection  string
	RKey        string
	CreatedAt   time.Time
	Title       string
	Rating      int
	HasRating   bool
	ChainDepth  int // 0 = fuzzy match, 1+ = sourceRef chain depth
}

type UsageGroup struct {
	Key           string
	Label         string
	Entries       []Entry
	Count         int
	RatingAverage float64
	RatingCount   int
	Page          int
	PerPage       int
	HasPrev       bool
	HasNext       bool
}

type Result struct {
	LibraryEntries []Entry
	LibraryCount   int
	Usage          []UsageGroup
	UsageCount     int
	RatingAverage  float64
	RatingCount    int
}

type IndexedRecord struct {
	URI        string
	DID        string
	Collection string
	RKey       string
	Record     json.RawMessage
	CreatedAt  time.Time
}

type Profile struct {
	Handle      string
	DisplayName string
	AvatarURL   string
}

type LookupOptions struct {
	UsageKey     string
	UsagePage    int
	UsagePerPage int
}

func (r *Result) IsEmpty() bool {
	return r == nil || (r.LibraryCount == 0 && r.UsageCount == 0)
}

type RecordSource interface {
	ListSourceRefChain(ctx context.Context, uri string, maxDepth, maxRecords int) ([]IndexedRecord, error)
	ListRecordsByCollectionOldest(ctx context.Context, collection string) ([]IndexedRecord, error)
	ListUsageBacklinks(ctx context.Context, uri, fromCollection, fieldName string) ([]IndexedRecord, error)
}

type ProfileResolver interface {
	GetProfile(ctx context.Context, did string) (*Profile, error)
}

type Service struct {
	src      RecordSource
	profiles ProfileResolver
}

func NewService(src RecordSource, profiles ProfileResolver) *Service {
	return &Service{src: src, profiles: profiles}
}

func (s *Service) Lookup(ctx context.Context, uri string) (*Result, error) {
	return s.LookupWithOptions(ctx, uri, LookupOptions{})
}

func (s *Service) LookupWithOptions(ctx context.Context, uri string, opts LookupOptions) (*Result, error) {
	if s == nil || s.src == nil || uri == "" {
		return &Result{}, nil
	}
	parsed, err := atp.ParseATURI(uri)
	if err != nil {
		return &Result{}, nil
	}
	cfg, ok := ConfigFor(parsed.Collection)
	if !ok {
		return &Result{}, nil
	}

	libByURI := map[string]Entry{}
	chain, err := s.src.ListSourceRefChain(ctx, uri, maxChainDepth, maxChainRecords)
	if err != nil {
		return nil, err
	}
	parentDepth := map[string]int{uri: 0}
	for _, rec := range chain {
		depth := 1
		if d, ok := parentDepth[sourceRefOf(rec.Record)]; ok {
			depth = d + 1
		}
		parentDepth[rec.URI] = depth
		libByURI[rec.URI] = entryFor(rec, depth)
	}

	if cfg.DedupKey != nil && len(cfg.AllFields) > 0 {
		s.addFuzzyMatches(ctx, uri, cfg, libByURI)
	}

	libEntries, libCount := dedupeLibraryByDID(libByURI)
	usageGroups, usageTotal, avgRating, ratingCount := s.usageBacklinks(ctx, uri, cfg, opts)

	profiles := s.resolveProfiles(ctx, uniqueDIDs(libEntries, usageGroups))
	for i := range libEntries {
		applyProfile(&libEntries[i], profiles)
	}
	for gi := range usageGroups {
		for ei := range usageGroups[gi].Entries {
			applyProfile(&usageGroups[gi].Entries[ei], profiles)
		}
	}

	return &Result{LibraryEntries: libEntries, LibraryCount: libCount, Usage: usageGroups, UsageCount: usageTotal, RatingAverage: avgRating, RatingCount: ratingCount}, nil
}

func (s *Service) addFuzzyMatches(ctx context.Context, uri string, cfg EntityConfig, libByURI map[string]Entry) {
	targetFields := fieldsForURI(ctx, s.src, uri, cfg)
	if targetFields == nil {
		return
	}
	targetKey := cfg.DedupKey(targetFields)
	if targetKey == "" {
		return
	}
	recs, err := s.src.ListRecordsByCollectionOldest(ctx, cfg.Collection)
	if err != nil {
		return
	}
	for _, rec := range recs {
		if rec.URI == uri {
			continue
		}
		if _, ok := libByURI[rec.URI]; ok {
			continue
		}
		fields := extractFields(cfg, rec.Record)
		if fields == nil || cfg.DedupKey(fields) != targetKey {
			continue
		}
		libByURI[rec.URI] = entryFor(rec, 0)
	}
}

func fieldsForURI(ctx context.Context, src RecordSource, uri string, cfg EntityConfig) map[string]string {
	recs, err := src.ListRecordsByCollectionOldest(ctx, cfg.Collection)
	if err != nil {
		return nil
	}
	for _, rec := range recs {
		if rec.URI == uri {
			return extractFields(cfg, rec.Record)
		}
	}
	return nil
}

func extractFields(cfg EntityConfig, record json.RawMessage) map[string]string {
	var data map[string]any
	if err := json.Unmarshal(record, &data); err != nil {
		return nil
	}
	fields := make(map[string]string, len(cfg.AllFields))
	for _, f := range cfg.AllFields {
		if v, ok := data[f].(string); ok && v != "" {
			fields[f] = v
		}
	}
	return fields
}

type pagedUsageSource interface {
	ListUsageBacklinksPage(ctx context.Context, uri, fromCollection, fieldName string, limit, offset int) ([]IndexedRecord, int, error)
}

type recordLookup interface {
	GetRecord(ctx context.Context, uri string) (IndexedRecord, bool)
}

func (s *Service) usageBacklinks(ctx context.Context, uri string, cfg EntityConfig, opts LookupOptions) ([]UsageGroup, int, float64, int) {
	groups := make([]UsageGroup, 0, len(cfg.UsageRefs))
	total := 0
	ratingTotal := 0
	ratingCount := 0
	for _, ref := range cfg.UsageRefs {
		key := ref.key()
		page, perPage := 1, 0
		if opts.UsagePerPage > 0 && (opts.UsageKey == "" || opts.UsageKey == key) {
			page = opts.UsagePage
			if page <= 0 {
				page = 1
			}
			perPage = opts.UsagePerPage
		}

		var recs []IndexedRecord
		var count int
		var err error
		if perPage > 0 {
			if paged, ok := s.src.(pagedUsageSource); ok {
				recs, count, err = paged.ListUsageBacklinksPage(ctx, uri, ref.Collection, ref.Field, perPage, (page-1)*perPage)
			} else {
				var all []IndexedRecord
				all, err = s.src.ListUsageBacklinks(ctx, uri, ref.Collection, ref.Field)
				count = len(all)
				start := (page - 1) * perPage
				if start < len(all) {
					end := min(start+perPage, len(all))
					recs = all[start:end]
				}
			}
		} else {
			recs, err = s.src.ListUsageBacklinks(ctx, uri, ref.Collection, ref.Field)
			count = len(recs)
		}
		if err != nil {
			continue
		}
		entries := make([]Entry, 0, len(recs))
		groupRatingTotal := 0
		groupRatingCount := 0
		for _, rec := range recs {
			e := entryFor(rec, 0)
			e.Title = titleFor(ctx, s.src, rec.Collection, rec.Record, rec.CreatedAt)
			if isBrewCollection(rec.Collection) {
				e.Rating, e.HasRating = ratingFor(rec.Record)
			}
			if e.HasRating {
				groupRatingTotal += e.Rating
				groupRatingCount++
			}
			entries = append(entries, e)
		}
		groupAvg, groupCount := s.usageRatingStats(ctx, uri, ref, groupRatingTotal, groupRatingCount)
		if groupCount > 0 {
			ratingTotal += int(math.Round(groupAvg * float64(groupCount)))
			ratingCount += groupCount
		}
		if perPage == 0 && len(entries) > maxEntries {
			entries = entries[:maxEntries]
		}
		total += count
		groups = append(groups, UsageGroup{Key: key, Label: ref.Label, Entries: entries, Count: count, RatingAverage: groupAvg, RatingCount: groupCount, Page: page, PerPage: perPage, HasPrev: perPage > 0 && page > 1, HasNext: perPage > 0 && page*perPage < count})
	}
	if ratingCount == 0 {
		return groups, total, 0, 0
	}
	return groups, total, float64(ratingTotal) / float64(ratingCount), ratingCount
}

func (s *Service) usageRatingStats(ctx context.Context, uri string, ref UsageRef, pageTotal, pageCount int) (float64, int) {
	all, err := s.src.ListUsageBacklinks(ctx, uri, ref.Collection, ref.Field)
	if err != nil {
		if pageCount == 0 {
			return 0, 0
		}
		return float64(pageTotal) / float64(pageCount), pageCount
	}
	total := 0
	count := 0
	for _, rec := range all {
		if !isBrewCollection(rec.Collection) {
			continue
		}
		if rating, ok := ratingFor(rec.Record); ok {
			total += rating
			count++
		}
	}
	if count == 0 {
		return 0, 0
	}
	return float64(total) / float64(count), count
}

func entryFor(rec IndexedRecord, depth int) Entry {
	return Entry{DID: rec.DID, RecordURI: rec.URI, Collection: rec.Collection, RKey: rec.RKey, CreatedAt: rec.CreatedAt, ChainDepth: depth}
}

func dedupeLibraryByDID(byURI map[string]Entry) ([]Entry, int) {
	byDID := map[string]Entry{}
	for _, e := range byURI {
		if existing, ok := byDID[e.DID]; !ok || e.CreatedAt.After(existing.CreatedAt) {
			byDID[e.DID] = e
		}
	}
	out := make([]Entry, 0, len(byDID))
	for _, e := range byDID {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	if len(out) > maxEntries {
		out = out[:maxEntries]
	}
	return out, len(byDID)
}

func (s *Service) resolveProfiles(ctx context.Context, dids []string) map[string]*Profile {
	out := make(map[string]*Profile, len(dids))
	if s.profiles == nil {
		return out
	}
	for _, did := range dids {
		if p, err := s.profiles.GetProfile(ctx, did); err == nil && p != nil {
			out[did] = p
		}
	}
	return out
}

func applyProfile(e *Entry, profiles map[string]*Profile) {
	p := profiles[e.DID]
	if p == nil {
		return
	}
	e.Handle = p.Handle
	e.DisplayName = p.DisplayName
	e.AvatarURL = p.AvatarURL
}

func uniqueDIDs(lib []Entry, usage []UsageGroup) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(did string) {
		if did == "" {
			return
		}
		if _, ok := seen[did]; ok {
			return
		}
		seen[did] = struct{}{}
		out = append(out, did)
	}
	for _, e := range lib {
		add(e.DID)
	}
	for _, g := range usage {
		for _, e := range g.Entries {
			add(e.DID)
		}
	}
	return out
}

func sourceRefOf(record json.RawMessage) string {
	var h struct {
		SourceRef string `json:"sourceRef"`
	}
	_ = json.Unmarshal(record, &h)
	return h.SourceRef
}

func ratingFor(record json.RawMessage) (int, bool) {
	var h struct {
		Rating *int `json:"rating"`
	}
	if err := json.Unmarshal(record, &h); err != nil || h.Rating == nil || *h.Rating <= 0 {
		return 0, false
	}
	return *h.Rating, true
}

func isBrewCollection(collection string) bool {
	return collection == "brew" || len(collection) >= len(".brew") && collection[len(collection)-len(".brew"):] == ".brew"
}

func titleFor(ctx context.Context, src RecordSource, collection string, record json.RawMessage, createdAt time.Time) string {
	var h struct {
		Name      string `json:"name"`
		Origin    string `json:"origin"`
		BeanRef   string `json:"beanRef"`
		BrewerRef string `json:"brewerRef"`
	}
	_ = json.Unmarshal(record, &h)
	if h.Name != "" {
		return h.Name
	}
	if h.Origin != "" {
		return h.Origin
	}
	if isBrewCollection(collection) && h.BrewerRef != "" {
		if title := titleForRef(ctx, src, h.BrewerRef); title != "" {
			return title
		}
	}
	if h.BeanRef != "" {
		if title := titleForRef(ctx, src, h.BeanRef); title != "" {
			return title
		}
	}
	if !createdAt.IsZero() {
		return "Brew · " + createdAt.Format("Jan 2")
	}
	return "Record"
}

func titleForRef(ctx context.Context, src RecordSource, ref string) string {
	lookup, ok := src.(recordLookup)
	if !ok {
		return ""
	}
	rec, found := lookup.GetRecord(ctx, ref)
	if !found {
		return ""
	}
	if title := titleFor(ctx, src, rec.Collection, rec.Record, rec.CreatedAt); title != "" && title != "Record" {
		return title
	}
	return ""
}
