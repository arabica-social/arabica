// Package entities provides a registry of descriptors for each Arabica record
// type. A descriptor captures the per-entity data that callers in feed, templ,
// handlers, and ogcard dispatch on, replacing scattered switch statements with
// a single lookup.
package entities

import (
	"fmt"
	"sort"
	"strings"

	"github.com/a-h/templ"

	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// Descriptor describes one Arabica record type.
type Descriptor struct {
	Type        lexicons.RecordType
	NSID        string
	DisplayName string // "Bean"
	Noun        string // "bean" — appears in copy: "added a new bean"
	URLPath     string // "beans" — share URLs and routes

	// FeedFilterLabel is the label shown on the feed filter pill for this
	// entity, e.g. "Beans". Empty means the entity is hidden from the
	// feed filter bar (used for reference entities like roaster that
	// rarely warrant a dedicated tab).
	FeedFilterLabel string

	// GetField extracts one named string field from a typed model pointer for
	// form prefill. Returns ("", false) if entity is nil or field is unknown.
	GetField func(entity any, field string) (string, bool)

	// RecordToModel converts a raw record map (as fetched from PDS or witness
	// cache) into the typed model for this entity. Returns the model as any;
	// callers type-assert. nil callback means the entity does not appear in
	// the feed pipeline.
	RecordToModel func(record map[string]any, uri string) (any, error)

	// RenderFeedContent returns the entity-specific clickable block for
	// feed.templ (anchor wrapper + entity content). The argument is a
	// *feed.FeedItem typed as any to avoid an import cycle (entities is
	// imported by feed via the descriptor registry). Callers in
	// internal/web/components/ type-assert. nil means the entity does
	// not render a content slot in the feed.
	RenderFeedContent func(item any) templ.Component

	// FeedCardCompact applies the .feed-card-compact CSS modifier to the
	// feed card wrapper. Used by entities with sparse content
	// (grinder/brewer/roaster).
	FeedCardCompact bool

	// EditURL returns the dedicated edit-page URL for an item, or "" if
	// the entity has no edit page (edited via modal on the manage page).
	// Item is *feed.FeedItem typed as any (see RenderFeedContent note).
	EditURL func(item any) string

	// EditModalURL returns the HTMX URL to load the entity's edit modal,
	// or "" if the entity has a dedicated edit page (EditURL) instead.
	// Item is *feed.FeedItem typed as any (see RenderFeedContent note).
	EditModalURL func(item any) string

	// RKey returns the record key of a typed record. The argument is the
	// concrete record pointer (e.g. *arabica.Bean) typed as any to avoid
	// import cycles. Returns "" if the assertion fails or the record is
	// nil. Used by feed.FeedItem.RKey() to build share/view URLs without
	// hard-coding each app's record types.
	RKey func(record any) string

	// DisplayTitle returns a human-readable title for a typed record
	// (used in share UI, OG cards, etc.). Empty string means "use the
	// descriptor's DisplayName as fallback." Special cases (e.g. brew
	// returns the bean's name) live in each app's implementation.
	DisplayTitle func(record any) string
}

var (
	registry  = map[lexicons.RecordType]*Descriptor{}
	nsidIndex = map[string]*Descriptor{}
)

// Register adds a descriptor. Called once per entity at package init.
// Panics on duplicate registration to catch wiring bugs at startup.
func Register(d *Descriptor) {
	if _, ok := registry[d.Type]; ok {
		panic(fmt.Sprintf("entities: duplicate descriptor for %s", d.Type))
	}
	registry[d.Type] = d
	nsidIndex[d.NSID] = d
}

// Get returns the descriptor for a record type, or nil if unregistered.
func Get(rt lexicons.RecordType) *Descriptor { return registry[rt] }

// GetByNSID returns the descriptor whose NSID matches, or nil if none.
// Used by the firehose feed pipeline which dispatches on collection NSID.
func GetByNSID(nsid string) *Descriptor { return nsidIndex[nsid] }

// GetByNoun returns the descriptor whose Noun matches, or nil if none.
// Used by the feed handler to map URL `?type=<noun>` to a RecordType for
// apps where Noun differs from the RecordType string (e.g. oolong's
// "tea" Noun maps to RecordType "oolong-tea").
func GetByNoun(noun string) *Descriptor {
	for _, d := range registry {
		if d.Noun == noun {
			return d
		}
	}
	return nil
}

// All returns descriptors in stable order (by RecordType). Use for route loops.
func All() []*Descriptor {
	out := make([]*Descriptor, 0, len(registry))
	for _, d := range registry {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}

// AllForApp returns descriptors whose NSID begins with `nsidBase + "."`,
// in stable RecordType order. Use this from per-app App constructors so
// that the global registry (which may hold descriptors from sister apps)
// doesn't leak across app boundaries.
func AllForApp(nsidBase string) []*Descriptor {
	prefix := nsidBase + "."
	out := make([]*Descriptor, 0, len(registry))
	for _, d := range registry {
		if strings.HasPrefix(d.NSID, prefix) {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}
