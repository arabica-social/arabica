package moderation

import (
	"context"

	"github.com/rs/zerolog/log"
)

// FilterSource provides the data needed to build a ContentFilter.
// Both moderation.Store and feed.ModerationFilter satisfy this interface.
type FilterSource interface {
	ListHiddenURIs(ctx context.Context) ([]string, error)
	ListBlacklistedDIDs(ctx context.Context) ([]string, error)
}

// ContentFilter holds pre-loaded moderation state for efficient per-item checks.
// Create one per request via LoadFilter, then use ShouldHide or FilterSlice.
type ContentFilter struct {
	hiddenURIs  map[string]bool
	blacklisted map[string]bool
}

// LoadFilter bulk-loads hidden URIs and blacklisted DIDs from the source (2 queries).
// Errors from the source are logged and degraded gracefully (partial filtering).
// A nil source returns an empty filter that hides nothing.
func LoadFilter(ctx context.Context, src FilterSource) (*ContentFilter, error) {
	f := &ContentFilter{
		hiddenURIs:  make(map[string]bool),
		blacklisted: make(map[string]bool),
	}

	if src == nil {
		return f, nil
	}

	if uris, err := src.ListHiddenURIs(ctx); err != nil {
		log.Warn().Err(err).Msg("moderation: failed to load hidden URIs for filter")
	} else {
		for _, uri := range uris {
			f.hiddenURIs[uri] = true
		}
	}

	if dids, err := src.ListBlacklistedDIDs(ctx); err != nil {
		log.Warn().Err(err).Msg("moderation: failed to load blacklisted DIDs for filter")
	} else {
		for _, did := range dids {
			f.blacklisted[did] = true
		}
	}

	return f, nil
}

// ShouldHide returns true if the record should be hidden, either because its
// URI is in the hidden set or its author DID is blacklisted.
// Empty strings are never matched.
func (f *ContentFilter) ShouldHide(uri, authorDID string) bool {
	if uri != "" && f.hiddenURIs[uri] {
		return true
	}
	if authorDID != "" && f.blacklisted[authorDID] {
		return true
	}
	return false
}

// IsBlocked returns true if the given DID is blacklisted.
func (f *ContentFilter) IsBlocked(did string) bool {
	return did != "" && f.blacklisted[did]
}

// FilterSlice removes items that should be hidden from a slice.
// The getKeys function extracts the AT-URI and author DID from each item.
// A nil filter returns the input unchanged.
func FilterSlice[T any](f *ContentFilter, items []T, getKeys func(T) (uri string, authorDID string)) []T {
	if f == nil {
		return items
	}

	result := make([]T, 0, len(items))
	for _, item := range items {
		uri, did := getKeys(item)
		if !f.ShouldHide(uri, did) {
			result = append(result, item)
		}
	}
	return result
}
