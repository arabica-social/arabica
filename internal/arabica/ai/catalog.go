package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
	"tangled.org/pdewey.com/atp"
)

// LibraryStore exposes no writes. Only the authenticated request's own store may
// be supplied by the handler; the browser cannot supply a library or account.
type LibraryStore interface {
	DID() string
	FetchAllRecords(context.Context, string) ([]records.RawRecord, error)
}
type libraryItem struct {
	ID                    string `json:"id"`
	Kind                  string `json:"kind"`
	Label                 string `json:"label"`
	BrewerType            string `json:"brewer_type,omitempty"`
	rkey                  string
	RequestMatch          bool `json:"request_match,omitempty"`
	RequestMatches        int  `json:"request_matches,omitempty"`
	SoleOpenDefault       bool `json:"sole_open_default,omitempty"`
	matchScore            int
	recentGround          bool
	pourCountMismatch     bool
	uniqueLabel           bool
	RoasterLabel          string         `json:"roaster,omitempty"`
	Closed                *bool          `json:"closed,omitempty"`
	OpenRoasterMatches    int            `json:"open_roaster_matches,omitempty"`
	BrewerLabel           string         `json:"brewer,omitempty"`
	Defaults              map[string]any `json:"defaults,omitempty"`
	Instructions          string         `json:"instructions,omitempty"`
	InstructionsTruncated bool           `json:"instructions_truncated,omitempty"`
	recipeBrewer          string
	score                 int
}
type catalog []libraryItem

var referenceKinds = map[string]string{"bean_rkey": "bean", "grinder_rkey": "grinder", "brewer_rkey": "brewer", "roaster_rkey": "roaster", "recipe_rkey": "recipe"}

func compact(s string, n int) string {
	if len(s) > n {
		s = s[:n]
		for !utf8.ValidString(s) {
			s = s[:len(s)-1]
		}
	}
	return s
}

// loadCatalog keeps provider context bounded while checking ambiguity against the
// complete authoritative read. The store's full materialization is not a completeness
// promise for the retained provider catalog.
func loadCatalog(ctx context.Context, store LibraryStore, draft Draft, query ...string) (catalog, error) {
	return loadCatalogPinned(ctx, store, draft, nil, query...)
}

func loadCatalogPinned(ctx context.Context, store LibraryStore, draft Draft, pins map[string]bool, query ...string) (catalog, error) {
	var result catalog
	// Dependency records are read first so joins never rely on model text or rkey alone.
	dependencies := map[string]map[string]libraryItem{}
	message := strings.Join(query, " ")
	terms := contextTerms(message)
	requestedPours := specifiedPourCount(message)
	for _, kind := range []string{"roaster", "brewer", "grinder", "bean", "recipe"} {
		nsid := arabica.NSIDBase + "." + kind
		rows, err := store.FetchAllRecords(ctx, nsid)
		if err != nil {
			return nil, ErrLibrary
		}
		selected, _ := draft.Fields[kind+"_rkey"].(string)
		found := selected == ""
		cap := 100
		if kind == "recipe" {
			cap = 20
		}
		for ref := range pins {
			if strings.HasPrefix(ref, "at://"+store.DID()+"/"+nsid+"/") {
				cap++
			}
		}
		chosen := make([]libraryItem, 0, cap)
		// Only dependency identity/labels need an index; bean and recipe history is
		// scanned without making another full-library projection.
		if kind == "roaster" || kind == "brewer" {
			dependencies[kind] = map[string]libraryItem{}
		}
		namedRoaster := false
		if kind == "bean" {
			for _, roaster := range dependencies["roaster"] {
				if exactMention(message, roaster.Label) {
					namedRoaster = true
					break
				}
			}
		}
		itemFor := func(row records.RawRecord) libraryItem {
			label, _ := row.Record["name"].(string)
			bt, _ := row.Record["brewerType"].(string)
			item := libraryItem{Kind: kind, Label: compact(label, 200), BrewerType: compact(bt, 100), rkey: row.RKey}
			if kind == "bean" {
				if value, exists := row.Record["closed"]; !exists {
					v := false
					item.Closed = &v
				} else if v, ok := value.(bool); ok {
					item.Closed = &v
				}
				ref, _ := row.Record["roasterRef"].(string)
				item.RoasterLabel = dependencies["roaster"][ref].Label
			}
			if kind == "recipe" {
				item.Defaults = recipeDefaults(row.Record)
				notes, _ := row.Record["notes"].(string)
				item.Instructions = compact(notes, 2000)
				item.InstructionsTruncated = len(notes) > len(item.Instructions)
				ref, _ := row.Record["brewerRef"].(string)
				brewer := dependencies["brewer"][ref]
				item.BrewerLabel, item.recipeBrewer = brewer.Label, brewer.rkey
			}
			if kind == "recipe" {
				item.matchScore = recipeIdentityScore(message, item.Label, item.BrewerLabel)
				knownPours := specifiedPourCount(item.Label)
				if knownPours == 0 {
					pours, _ := item.Defaults["pours"].([]any)
					knownPours = len(pours)
				}
				if requestedPours > 0 && knownPours > 0 {
					// A fully named source can be intentionally adapted; a shorthand
					// count instead distinguishes which saved source the user means.
					item.pourCountMismatch = requestedPours != knownPours && !exactMention(message, item.Label)
					if item.pourCountMismatch {
						item.matchScore = 0
					} else if requestedPours == knownPours {
						if item.matchScore == 0 && identityScore(message, item.BrewerLabel) > 0 {
							item.matchScore = 1
						}
						if item.matchScore > 0 {
							item.matchScore += 4
						}
					}
				}
			} else {
				item.matchScore = identityScore(message, item.Label)
				if kind == "bean" {
					if normalized(item.Label) == normalized(item.RoasterLabel) {
						item.matchScore = 0
					}
					if namedRoaster && !exactMention(message, item.RoasterLabel) {
						item.matchScore = 0
					}
					if item.matchScore == 0 && exactMention(message, item.RoasterLabel) {
						item.matchScore = 1
					}
				}
			}
			if exactMention(message, item.Label) || exactMention(message, item.RoasterLabel) || exactMention(message, item.BrewerLabel) {
				item.score += 10
			}
			item.score += contextRelevance(terms, item.Label, item.RoasterLabel, item.BrewerLabel)
			if item.Closed != nil && !*item.Closed {
				item.score++
			}
			if pins["at://"+store.DID()+"/"+nsid+"/"+row.RKey] {
				item.score += 1000
			}
			if row.RKey == selected {
				item.score += 100
				found = true
			}
			return item
		}
		bestMatch, matchCount, openCount, lexicalBeans := 0, 0, 0, 0
		for _, row := range rows {
			if !atp.ValidateRKey(row.RKey) || (row.URI != "" && row.URI != "at://"+store.DID()+"/"+nsid+"/"+row.RKey) {
				return nil, ErrLibrary
			}
			item := itemFor(row)
			if dep := dependencies[kind]; dep != nil {
				dep["at://"+store.DID()+"/"+nsid+"/"+row.RKey] = item
			}
			duplicate := false
			for _, prev := range chosen {
				if prev.rkey == item.rkey {
					duplicate = true
					break
				}
			}
			if duplicate {
				continue
			}
			eligible := kind != "bean" || item.Closed != nil && !*item.Closed
			if kind == "bean" {
				if item.matchScore > 0 {
					lexicalBeans++
				}
				if eligible {
					openCount++
				}
			}
			if eligible && item.matchScore > 0 {
				if item.matchScore > bestMatch {
					bestMatch, matchCount = item.matchScore, 0
				}
				if item.matchScore == bestMatch {
					matchCount++
				}
			}
			// Keep the same identity candidates that the full-row count describes.
			item.score += item.matchScore * 10
			if len(chosen) < cap {
				chosen = append(chosen, item)
				continue
			}
			// Selection is an invariant, not a relevance score. Never evict
			// the current reference, and always retain it when encountered late.
			lowest := -1
			for i := range chosen {
				if chosen[i].rkey == selected || pins["at://"+store.DID()+"/"+nsid+"/"+chosen[i].rkey] {
					continue
				}
				if lowest < 0 || chosen[i].score < chosen[lowest].score {
					lowest = i
				}
			}
			if lowest >= 0 && (item.rkey == selected || pins["at://"+store.DID()+"/"+nsid+"/"+item.rkey] || item.score > chosen[lowest].score) {
				chosen[lowest] = item
			}
		}
		if !found {
			return nil, ErrLibrary
		}
		counts, roasterCounts := map[string]int{}, map[string]int{}
		for _, item := range chosen {
			counts[normalized(item.Label)] = 0
			if item.RoasterLabel != "" {
				roasterCounts[normalized(item.RoasterLabel)] = 0
			}
		}
		for _, row := range rows {
			item := itemFor(row)
			if _, ok := counts[normalized(item.Label)]; ok {
				counts[normalized(item.Label)]++
			}
			if _, ok := roasterCounts[normalized(item.RoasterLabel)]; ok && item.Closed != nil && !*item.Closed {
				roasterCounts[normalized(item.RoasterLabel)]++
			}
		}
		for i := range chosen {
			eligible := kind != "bean" || chosen[i].Closed != nil && !*chosen[i].Closed
			if eligible && bestMatch > 0 && chosen[i].matchScore == bestMatch {
				chosen[i].RequestMatches = matchCount
				chosen[i].RequestMatch = matchCount == 1
			}
			chosen[i].SoleOpenDefault = kind == "bean" && eligible && openCount == 1 && lexicalBeans == 0 && !namedRoaster && selected == "" && draft.Kind == "brew"
			chosen[i].uniqueLabel = counts[normalized(chosen[i].Label)] == 1
			chosen[i].OpenRoasterMatches = roasterCounts[normalized(chosen[i].RoasterLabel)]
			chosen[i].ID = fmt.Sprintf("%s-%d", kind, i+1)
		}
		result = append(result, chosen...)
	}
	// Only retained own brewer references may become provider IDs or write defaults.
	for i := range result {
		if result[i].Kind != "recipe" {
			continue
		}
		for _, brewer := range result {
			if brewer.Kind == "brewer" && brewer.rkey == result[i].recipeBrewer {
				result[i].Defaults["brewer_rkey"] = brewer.ID
				break
			}
		}
	}
	return result, nil
}

// recordNumber tolerates the raw-store numeric representations without accepting
// strings or fractions where the lexicon requires persisted integers.
func recordNumber(value any) (float64, bool) {
	var n float64
	switch v := value.(type) {
	case float64:
		n = v
	case int:
		n = float64(v)
	case int64:
		n = float64(v)
	case json.Number:
		var err error
		n, err = v.Float64()
		if err != nil {
			return 0, false
		}
	default:
		return 0, false
	}
	return n, !math.IsNaN(n) && !math.IsInf(n, 0) && n >= 0 && n <= maxSafeNumber && math.Trunc(n) == n
}

func recipeDefaults(record map[string]any) map[string]any {
	out := map[string]any{}
	for raw, field := range map[string]string{"coffeeAmount": "coffee_amount", "waterAmount": "water_amount"} {
		if n, ok := recordNumber(record[raw]); ok {
			n /= 10
			// Context preserves fractional amounts; recipePatch separately enforces the
			// current integer brew API and never silently rounds a dose.
			if n <= fields["brew"][field].Max {
				out[field] = n
			}
		}
	}
	if pours, ok := record["pours"].([]any); ok && len(pours) <= 100 {
		normalized := make([]any, 0, len(pours))
		valid := true
		for _, raw := range pours {
			pour, ok := raw.(map[string]any)
			if !ok {
				valid = false
				break
			}
			water, wok := recordNumber(pour["waterAmount"])
			seconds, sok := recordNumber(pour["timeSeconds"])
			if !wok || !sok {
				valid = false
				break
			}
			normalized = append(normalized, map[string]any{"water_amount": water, "time_seconds": seconds})
		}
		if valid {
			out["pours"] = normalized
		}
	}
	return out
}

// recipePatch returns only application-derived fields safe for the current brew
// mutation API. Recipe instructions are context, never brew tasting notes.
func (c catalog) recipePatch(rkey string) (map[string]any, string, error) {
	for _, item := range c {
		if item.Kind != "recipe" || item.rkey != rkey {
			continue
		}
		patch := map[string]any{"recipe_rkey": rkey}
		category := ""
		for key, value := range item.Defaults {
			if key == "brewer_rkey" {
				for _, brewer := range c {
					if brewer.Kind == "brewer" && brewer.ID == value {
						patch[key] = brewer.rkey
						category = brewer.BrewerType
						break
					}
				}
			} else if validateFields("brew", map[string]any{key: value}) == nil {
				patch[key] = value
			}
		}
		return patch, category, nil
	}
	return nil, "", errFields
}

func (c catalog) convert(patch map[string]any, toProvider bool) (map[string]any, string, error) {
	out := make(map[string]any, len(patch))
	brewerType := ""
	for key, value := range patch {
		kind, isRef := referenceKinds[key]
		if !isRef || value == nil || value == "" {
			out[key] = value
			continue
		}
		s, ok := value.(string)
		if !ok {
			return nil, "", errFields
		}
		found := false
		for _, item := range c {
			match := item.ID
			if toProvider {
				match = item.rkey
			}
			if item.Kind == kind && s == match {
				if toProvider {
					out[key] = item.ID
				} else {
					out[key] = item.rkey
				}
				if kind == "brewer" {
					brewerType = item.BrewerType
				}
				found = true
				break
			}
		}
		if !found {
			return nil, "", errFields
		}
	}
	return out, brewerType, nil
}

// deriveLabel makes library choice text authoritative instead of trusting a
// model label that might misdescribe the real linked record.
func (c catalog) deriveLabel(patch map[string]any) string {
	var labels []string
	for _, key := range []string{"bean_rkey", "grinder_rkey", "brewer_rkey", "roaster_rkey", "recipe_rkey"} {
		if rkey, ok := patch[key].(string); ok {
			for _, item := range c {
				if item.Kind == referenceKinds[key] && item.rkey == rkey {
					label := item.Label
					if item.Kind == "bean" && item.RoasterLabel != "" {
						label += " · " + item.RoasterLabel
					}
					if item.Kind == "bean" && item.Closed != nil && *item.Closed {
						label += " (closed)"
					}
					if item.Kind == "recipe" && item.BrewerLabel != "" {
						label += " · " + item.BrewerLabel
					}
					labels = append(labels, label)
					break
				}
			}
		}
	}
	return strings.Join(labels, " · ")
}
