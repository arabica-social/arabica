// Package entities provides registries for record identity metadata and
// record-specific behavior. Descriptors identify records; record behavior
// owns codecs, rkey/title extraction, form field extraction, and feed ref
// hydration.
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
