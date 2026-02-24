package suggestions

import (
	"encoding/json"
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
}

// REFACTOR: this should be able to use the structs probably
var entityConfigs = map[string]entityFieldConfig{
	atproto.NSIDRoaster: {
		allFields:    []string{"name", "location", "website"},
		searchFields: []string{"name", "location", "website"},
		nameField:    "name",
	},
	atproto.NSIDGrinder: {
		allFields:    []string{"name", "grinderType", "burrType"},
		searchFields: []string{"name", "grinderType", "burrType"},
		nameField:    "name",
	},
	atproto.NSIDBrewer: {
		allFields:    []string{"name", "brewerType", "description"},
		searchFields: []string{"name", "brewerType"},
		nameField:    "name",
	},
	atproto.NSIDBean: {
		allFields:    []string{"name", "origin", "roastLevel", "process"},
		searchFields: []string{"name", "origin", "roastLevel"},
		nameField:    "name",
	},
}

// Search searches indexed records for entity suggestions matching a query.
// It matches against searchable fields, deduplicates by normalized name,
// and returns results sorted by popularity.
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

		// Deduplicate by normalized name
		normalizedName := strings.ToLower(strings.TrimSpace(name))

		if existing, ok := candidates[normalizedName]; ok {
			existing.dids[indexed.DID] = struct{}{}
			// Keep the record with more complete fields
			nonEmpty := 0
			for _, v := range fields {
				if v != "" {
					nonEmpty++
				}
			}
			if nonEmpty > existing.fieldCount {
				existing.suggestion.Name = name
				existing.suggestion.Fields = fields
				existing.suggestion.SourceURI = indexed.URI
				existing.fieldCount = nonEmpty
			}
		} else {
			nonEmpty := 0
			for _, v := range fields {
				if v != "" {
					nonEmpty++
				}
			}
			candidates[normalizedName] = &candidate{
				suggestion: EntitySuggestion{
					Name:      name,
					SourceURI: indexed.URI,
					Fields:    fields,
				},
				fieldCount: nonEmpty,
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
