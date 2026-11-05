package suggestions

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
)

// EntitySuggestion represents a suggestion for auto-completing an entity
type EntitySuggestion struct {
	Name      string            `json:"name"`
	SourceURI string            `json:"source_uri"`
	Fields    map[string]string `json:"fields"`
	Count     int               `json:"count"`
}

// IndexedRecord is the projection of a stored record that the
// suggestion engine cares about. Defined here (rather than imported
// from firehose) so the suggestions package stays free of
// app-specific transitive imports.
type IndexedRecord struct {
	URI    string
	DID    string
	Record []byte
}

// RecordSource provides read access to indexed records.
type RecordSource interface {
	ListRecordsByCollectionOldest(ctx context.Context, collection string) ([]IndexedRecord, error)
	CountReferencesToURI(ctx context.Context, uri string) (int, error)
}

// PreferredDIDs is a set of DIDs whose records should be preferred when
// choosing the representative sourceRef for a suggestion. Records from
// preferred DIDs get a scoring bonus during deduplication.
var PreferredDIDs = map[string]struct{}{}

type FieldConfig struct {
	AllFields    []string
	SearchFields []string
	NameField    string
	DedupKey     func(fields map[string]string) string
	Prepare      func(ctx context.Context, source RecordSource) any
	Enrich       func(prepared any, record map[string]any, fields map[string]string)
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

	// Run the config-provided Prepare hook once. Apps use this to load
	// cross-collection data (e.g. arabica's bean config builds a
	// roaster URI → name map here) without the suggestions package
	// itself needing to know any collection NSIDs.
	var prepared any
	if config.Prepare != nil {
		prepared = config.Prepare(ctx, source)
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
		for _, f := range config.AllFields {
			if v, ok := recordData[f].(string); ok && v != "" {
				fields[f] = v
			}
		}

		if config.Enrich != nil {
			config.Enrich(prepared, recordData, fields)
		}

		name := fields[config.NameField]
		if name == "" {
			continue
		}

		// Check if any searchable field matches the query
		matched := false
		for _, sf := range config.SearchFields {
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
		key := config.DedupKey(fields)

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

// BuildNameMap loads every record of `nsid` indexed locally and
// returns an AT-URI → record-`name` map. Configs use this in their
// Prepare hook to resolve cross-collection references (e.g. resolving
// a bean's `roasterRef` to the roaster's name).
func BuildNameMap(ctx context.Context, source RecordSource, nsid string) map[string]string {
	records, err := source.ListRecordsByCollectionOldest(ctx, nsid)
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
