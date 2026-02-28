package suggestions

import (
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	"arabica/internal/atproto"
	"arabica/internal/firehose"
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
	ListRecordsByCollection(collection string) ([]firehose.IndexedRecord, error)
}

// entityFieldConfig defines which fields to extract and search for each entity type
type entityFieldConfig struct {
	allFields    []string
	searchFields []string
	nameField    string
	dedupKey     func(fields map[string]string) string
}

var entityConfigs = map[string]entityFieldConfig{
	atproto.NSIDRoaster: {
		allFields:    []string{"name", "location", "website"},
		searchFields: []string{"name", "location", "website"},
		nameField:    "name",
		dedupKey:     roasterDedupKey,
	},
	atproto.NSIDGrinder: {
		allFields:    []string{"name", "grinderType", "burrType"},
		searchFields: []string{"name", "grinderType", "burrType"},
		nameField:    "name",
		dedupKey:     grinderDedupKey,
	},
	atproto.NSIDBrewer: {
		allFields:    []string{"name", "brewerType", "description"},
		searchFields: []string{"name", "brewerType"},
		nameField:    "name",
		dedupKey:     brewerDedupKey,
	},
	atproto.NSIDBean: {
		allFields:    []string{"name", "origin", "roastLevel", "process"},
		searchFields: []string{"name", "origin", "roastLevel"},
		nameField:    "name",
		dedupKey:     beanDedupKey,
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
func Search(source RecordSource, collection, query string, limit int) ([]EntitySuggestion, error) {
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

	records, err := source.ListRecordsByCollection(collection)
	if err != nil {
		return nil, err
	}

	// dedupKey -> aggregated suggestion
	type candidate struct {
		suggestion EntitySuggestion
		fieldCount int // number of non-empty fields (to pick best representative)
		dids       map[string]struct{}
	}
	candidates := make(map[string]*candidate)

	for _, indexed := range records {
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

		if existing, ok := candidates[key]; ok {
			existing.dids[indexed.DID] = struct{}{}
			// Keep the record with more complete fields
			nonEmpty := countNonEmpty(fields)
			if nonEmpty > existing.fieldCount {
				existing.suggestion.Name = name
				existing.suggestion.Fields = fields
				existing.suggestion.SourceURI = indexed.URI
				existing.fieldCount = nonEmpty
			}
		} else {
			candidates[key] = &candidate{
				suggestion: EntitySuggestion{
					Name:      name,
					SourceURI: indexed.URI,
					Fields:    fields,
				},
				fieldCount: countNonEmpty(fields),
				dids:       map[string]struct{}{indexed.DID: {}},
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

func countNonEmpty(fields map[string]string) int {
	n := 0
	for _, v := range fields {
		if v != "" {
			n++
		}
	}
	return n
}
