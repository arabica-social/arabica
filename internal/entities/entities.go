// Package entities provides a registry of descriptors for each Arabica record
// type. A descriptor captures the per-entity data that callers in feed, templ,
// handlers, and ogcard dispatch on, replacing scattered switch statements with
// a single lookup.
package entities

import (
	"fmt"
	"sort"

	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// Descriptor describes one Arabica record type.
type Descriptor struct {
	Type        lexicons.RecordType
	NSID        string
	DisplayName string // "Bean"
	Noun        string // "bean" — appears in copy: "added a new bean"
	URLPath     string // "beans" — share URLs and routes

	// GetField extracts one named string field from a typed model pointer for
	// form prefill. Returns ("", false) if entity is nil or field is unknown.
	GetField func(entity any, field string) (string, bool)
}

var registry = map[lexicons.RecordType]*Descriptor{}

// Register adds a descriptor. Called once per entity at package init.
// Panics on duplicate registration to catch wiring bugs at startup.
func Register(d *Descriptor) {
	if _, ok := registry[d.Type]; ok {
		panic(fmt.Sprintf("entities: duplicate descriptor for %s", d.Type))
	}
	registry[d.Type] = d
}

// Get returns the descriptor for a record type, or nil if unregistered.
func Get(rt lexicons.RecordType) *Descriptor { return registry[rt] }

// All returns descriptors in stable order (by RecordType). Use for route loops.
func All() []*Descriptor {
	out := make([]*Descriptor, 0, len(registry))
	for _, d := range registry {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Type < out[j].Type })
	return out
}
