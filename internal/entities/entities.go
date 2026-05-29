// Package entities provides a registry of descriptors for each record type.
// A descriptor captures domain and codec behavior for records. App-owned
// packages own route paths, UI labels, and rendering metadata.
package entities

import (
	"fmt"
	"sort"
	"strings"

	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// Descriptor describes one Arabica record type.
type Descriptor struct {
	Type        lexicons.RecordType
	NSID        string
	DisplayName string // "Bean"

	// GetField extracts one named string field from a typed model pointer for
	// form prefill. Returns ("", false) if entity is nil or field is unknown.
	GetField func(entity any, field string) (string, bool)

	// RecordToModel converts a raw record map (as fetched from PDS or witness
	// cache) into the typed model for this entity. Returns the model as any;
	// callers type-assert. nil callback means the entity does not appear in
	// the feed pipeline.
	RecordToModel func(record map[string]any, uri string) (any, error)

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

	// ResolveRefs hydrates cross-entity references on a typed model using
	// records already pulled from the firehose index. model is the typed
	// pointer returned by RecordToModel; recordData is the same raw map
	// the model was decoded from (used to read foreign ref URIs);
	// lookup returns a pre-decoded record map for a given AT-URI, or
	// (nil, false) if not in the batch. Implementations should silently
	// skip missing refs — feed cards render fine with partial data.
	// nil means the entity has no cross-record references to resolve.
	ResolveRefs func(model any, recordData map[string]any, lookup func(refURI string) (map[string]any, bool))
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
