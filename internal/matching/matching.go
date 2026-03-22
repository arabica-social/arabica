// Package matching provides generic entity matching across user records.
//
// When copying or forking records between users, references to entities (brewers,
// beans, grinders, etc.) need to be resolved to equivalent entities in the
// target user's collection. This package provides a reusable matching pipeline
// that can be extended with fuzzy matching in the future.
package matching

import "strings"

// Candidate represents a matchable entity from the user's records.
type Candidate struct {
	RKey string
	Name string
	Type string // optional secondary field (e.g. brewer_type)
}

// Result represents a matched entity with a confidence score.
type Result struct {
	RKey  string
	Name  string
	Score float64 // 1.0 = exact name match, 0.5 = type-only match
}

// Match finds the best matching candidate for the given source entity.
// It applies matchers in priority order and returns the first match, or nil.
//
// Current matching strategy (in priority order):
//  1. Exact name match (case-insensitive)
//  2. Single type match — if exactly one candidate shares the same type
//
// Either sourceName or sourceType may be empty; matching adapts accordingly.
func Match(sourceName, sourceType string, candidates []Candidate) *Result {
	if len(candidates) == 0 {
		return nil
	}

	// 1. Exact name match (case-insensitive)
	if sourceName != "" {
		for _, c := range candidates {
			if strings.EqualFold(c.Name, sourceName) {
				return &Result{RKey: c.RKey, Name: c.Name, Score: 1.0}
			}
		}
	}

	// 2. Single type match — only when there's exactly one candidate of the same type
	if sourceType != "" {
		var typeMatches []Candidate
		for _, c := range candidates {
			if c.Type != "" && strings.EqualFold(c.Type, sourceType) {
				typeMatches = append(typeMatches, c)
			}
		}
		if len(typeMatches) == 1 {
			return &Result{RKey: typeMatches[0].RKey, Name: typeMatches[0].Name, Score: 0.5}
		}
	}

	return nil
}
