package suggestions

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/firehose"
)

// EntitySuggestion represents a suggestion for auto-completing an entity
type EntitySuggestion struct {
	Name      string            `json:"name"`
	SourceURI string            `json:"source_uri"`
	Fields    map[string]string `json:"fields"`
	Count     int               `json:"count"`
}

// RecordSource provides read access to indexed records.
type RecordSource interface {
	ListRecordsByCollectionOldest(ctx context.Context, collection string) ([]firehose.IndexedRecord, error)
	CountReferencesToURI(ctx context.Context, uri string) (int, error)
}

// PreferredDIDs is a set of DIDs whose records should be preferred when
// choosing the representative sourceRef for a suggestion. Records from
// preferred DIDs get a scoring bonus during deduplication.
var PreferredDIDs = map[string]struct{}{}

// entityFieldConfig defines which fields to extract and search for each entity type
type entityFieldConfig struct {
	allFields    []string
	searchFields []string
	nameField    string
	dedupKey     func(fields map[string]string) string
}

var entityConfigs = map[string]entityFieldConfig{
	arabica.NSIDRoaster: {
		allFields:    []string{"name", "location", "website"},
		searchFields: []string{"name", "location", "website"},
		nameField:    "name",
		dedupKey:     roasterDedupKey,
	},
	arabica.NSIDGrinder: {
		allFields:    []string{"name", "grinderType", "burrType"},
		searchFields: []string{"name", "grinderType", "burrType"},
		nameField:    "name",
		dedupKey:     grinderDedupKey,
	},
	arabica.NSIDBrewer: {
		allFields:    []string{"name", "brewerType", "description"},
		searchFields: []string{"name", "brewerType"},
		nameField:    "name",
		dedupKey:     brewerDedupKey,
	},
	arabica.NSIDBean: {
		allFields:    []string{"name", "origin", "roastLevel", "process"},
		searchFields: []string{"name", "origin", "roastLevel"},
		nameField:    "name",
		dedupKey:     beanDedupKey,
	},
	arabica.NSIDRecipe: {
		allFields:    []string{"name", "brewerType"},
		searchFields: []string{"name", "brewerType"},
		nameField:    "name",
		dedupKey:     recipeDedupKey,
	},
}

// --- Dedup key functions ---
// Each returns a string that groups "same entity" records together.
// Records with the same dedup key are merged; different keys stay separate.

// roasterDedupKey: fuzzy name + normalized location.
// "Counter Culture Coffee" in "Durham, NC" vs "Counter Culture" in "Durham" → same.
// "Stumptown" in "Portland" vs "Stumptown" in "NYC" → different.
// Website is not included because it's too sparse — many records lack one,
// causing false splits. Website is still kept in Fields for display.
func roasterDedupKey(fields map[string]string) string {
	parts := []string{fuzzyName(fields["name"])}
	if loc := normalize(fields["location"]); loc != "" {
		parts = append(parts, loc)
	}
	return strings.Join(parts, "|")
}

// grinderDedupKey: exact name + grinder type + burr type.
// "1Zpresso JX Pro" hand/conical vs "1Zpresso JX Pro" electric/flat → different.
func grinderDedupKey(fields map[string]string) string {
	parts := []string{normalize(fields["name"])}
	if gt := normalize(fields["grinderType"]); gt != "" {
		parts = append(parts, gt)
	}
	if bt := normalize(fields["burrType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}

// brewerDedupKey: exact name + brewer type.
// "Hario V60" pour-over vs "Hario V60" dripper → different (if someone miscategorized).
func brewerDedupKey(fields map[string]string) string {
	parts := []string{normalize(fields["name"])}
	if bt := normalize(fields["brewerType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}

// beanDedupKey: exact name + origin + process.
// "Yirgacheffe" from Ethiopia/washed vs "Yirgacheffe" from Ethiopia/natural → different.
func beanDedupKey(fields map[string]string) string {
	parts := []string{normalize(fields["name"])}
	if o := normalize(fields["origin"]); o != "" {
		parts = append(parts, o)
	}
	if p := normalize(fields["process"]); p != "" {
		parts = append(parts, p)
	}
	return strings.Join(parts, "|")
}

// recipeDedupKey: exact name + brewer type.
// "V60 Standard" pourover vs "V60 Standard" immersion → different.
func recipeDedupKey(fields map[string]string) string {
	parts := []string{normalize(fields["name"])}
	if bt := normalize(fields["brewerType"]); bt != "" {
		parts = append(parts, bt)
	}
	return strings.Join(parts, "|")
}

// --- Normalization helpers ---

// normalize lowercases, trims whitespace, and collapses internal whitespace.
func normalize(s string) string {
	return collapseSpaces(strings.ToLower(strings.TrimSpace(s)))
}

// Common suffixes stripped during fuzzy name normalization for roasters/brewers.
// Order matters: longer suffixes first to avoid partial stripping.
var commonSuffixes = []string{
	"coffee roasters",
	"coffee roasting",
	"coffee company",
	"coffee co",
	"roasting company",
	"roasting co",
	"roasters",
	"roasting",
	"coffee",
	"co.",
}

// fuzzyName normalizes a name by lowercasing, stripping common coffee-industry
// suffixes, punctuation, and extra whitespace. This lets "Counter Culture Coffee"
// and "Counter Culture" merge, while still keeping genuinely different names apart.
func fuzzyName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))

	// Strip common suffixes
	for _, suffix := range commonSuffixes {
		if strings.HasSuffix(s, suffix) {
			s = strings.TrimSpace(s[:len(s)-len(suffix)])
			break // only strip one suffix
		}
	}

	// Remove punctuation (keep letters, digits, spaces)
	s = stripPunctuation(s)

	return collapseSpaces(s)
}

var nonAlphanumSpace = regexp.MustCompile(`[^a-z0-9\s]`)

func stripPunctuation(s string) string {
	return nonAlphanumSpace.ReplaceAllString(s, "")
}

var multiSpace = regexp.MustCompile(`\s+`)

func collapseSpaces(s string) string {
	return strings.TrimSpace(multiSpace.ReplaceAllString(s, " "))
}

// extractDomain pulls the domain from a URL for normalization.
// "https://www.counterculturecoffee.com/shop" → "counterculturecoffee.com"
func extractDomain(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	s := strings.ToLower(strings.TrimSpace(rawURL))
	// Strip scheme
	if i := strings.Index(s, "://"); i >= 0 {
		s = s[i+3:]
	}
	// Strip www.
	s = strings.TrimPrefix(s, "www.")
	// Strip path
	if i := strings.IndexByte(s, '/'); i >= 0 {
		s = s[:i]
	}
	// Strip port
	if i := strings.IndexByte(s, ':'); i >= 0 {
		s = s[:i]
	}
	return s
}

// Search searches indexed records for entity suggestions matching a query.
// It matches against searchable fields, deduplicates using entity-specific
// keys, and returns results sorted by popularity.
// If excludeDID is non-empty, records from that DID are excluded from results.
func Search(ctx context.Context, source RecordSource, collection, query string, limit int, excludeDID ...string) ([]EntitySuggestion, error) {
	if limit <= 0 {
		limit = 10
	}

	config, ok := entityConfigs[collection]
	if !ok {
		return nil, nil
	}

	queryLower := strings.ToLower(strings.TrimSpace(query))
	if len(queryLower) < 2 {
		return nil, nil
	}

	var skipDID string
	if len(excludeDID) > 0 {
		skipDID = excludeDID[0]
	}

	records, err := source.ListRecordsByCollectionOldest(ctx, collection)
	if err != nil {
		return nil, err
	}

	// For beans, build a roaster URI -> name map so we can include the
	// roaster name in suggestion fields. This lets clients verify the
	// roaster matches before setting source_ref.
	var roasterNames map[string]string
	if collection == arabica.NSIDBean {
		roasterNames = buildRoasterNameMap(ctx, source)
	}

	// dedupKey -> aggregated suggestion
	type candidate struct {
		suggestion EntitySuggestion
		score      int // composite score for picking best representative
		dids       map[string]struct{}
	}
	candidates := make(map[string]*candidate)

	for _, indexed := range records {
		// Skip the current user's records so they only see community data
		if skipDID != "" && indexed.DID == skipDID {
			continue
		}

		var recordData map[string]any
		if err := json.Unmarshal(indexed.Record, &recordData); err != nil {
			continue
		}

		// Extract fields
		fields := make(map[string]string)
		for _, f := range config.allFields {
			if v, ok := recordData[f].(string); ok && v != "" {
				fields[f] = v
			}
		}

		// For beans, resolve roasterRef to a roaster name
		if roasterNames != nil {
			if ref, ok := recordData["roasterRef"].(string); ok && ref != "" {
				if rn, ok := roasterNames[ref]; ok {
					fields["roasterName"] = rn
				}
			}
		}

		name := fields[config.nameField]
		if name == "" {
			continue
		}

		// Check if any searchable field matches the query
		matched := false
		for _, sf := range config.searchFields {
			val := strings.ToLower(fields[sf])
			if val == "" {
				continue
			}
			if strings.HasPrefix(val, queryLower) || strings.Contains(val, queryLower) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		// Deduplicate using entity-specific key
		key := config.dedupKey(fields)

		score := scoreRecord(ctx, source, indexed.URI, indexed.DID, fields)

		if existing, ok := candidates[key]; ok {
			existing.dids[indexed.DID] = struct{}{}
			if score > existing.score {
				existing.suggestion.Name = name
				existing.suggestion.Fields = fields
				existing.suggestion.SourceURI = indexed.URI
				existing.score = score
			}
		} else {
			candidates[key] = &candidate{
				suggestion: EntitySuggestion{
					Name:      name,
					SourceURI: indexed.URI,
					Fields:    fields,
				},
				score: score,
				dids:  map[string]struct{}{indexed.DID: {}},
			}
		}
	}

	// Build results with counts
	results := make([]EntitySuggestion, 0, len(candidates))
	for _, c := range candidates {
		c.suggestion.Count = len(c.dids)
		results = append(results, c.suggestion)
	}

	// Sort: prefix matches first, then by count desc, then alphabetically
	sort.Slice(results, func(i, j int) bool {
		iPrefix := strings.HasPrefix(strings.ToLower(results[i].Name), queryLower)
		jPrefix := strings.HasPrefix(strings.ToLower(results[j].Name), queryLower)
		if iPrefix != jPrefix {
			return iPrefix
		}
		if results[i].Count != results[j].Count {
			return results[i].Count > results[j].Count
		}
		return strings.ToLower(results[i].Name) < strings.ToLower(results[j].Name)
	})

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// scoreRecord computes a composite score for choosing the best representative
// record within a dedup group. Higher score wins. Factors:
//   - Field completeness (1 point per non-empty field)
//   - Reference count (2 points per record that references this URI via sourceRef)
//   - Preferred DID bonus (10 points if the record's author is in PreferredDIDs)
func scoreRecord(ctx context.Context, source RecordSource, uri, did string, fields map[string]string) int {
	score := countNonEmpty(fields)

	if refCount, err := source.CountReferencesToURI(ctx, uri); err == nil {
		score += refCount * 2
	}

	if _, ok := PreferredDIDs[did]; ok {
		score += 10
	}

	return score
}

// buildRoasterNameMap loads all indexed roaster records and returns a map
// from AT-URI to roaster name. Used to resolve roaster references in bean
// suggestions so the client can verify roaster match before setting source_ref.
func buildRoasterNameMap(ctx context.Context, source RecordSource) map[string]string {
	records, err := source.ListRecordsByCollectionOldest(ctx, arabica.NSIDRoaster)
	if err != nil {
		return nil
	}
	m := make(map[string]string, len(records))
	for _, r := range records {
		var data map[string]any
		if err := json.Unmarshal(r.Record, &data); err != nil {
			continue
		}
		if name, ok := data["name"].(string); ok && name != "" {
			m[r.URI] = name
		}
	}
	return m
}

func countNonEmpty(fields map[string]string) int {
	n := 0
	for _, v := range fields {
		if v != "" {
			n++
		}
	}
	return n
}
