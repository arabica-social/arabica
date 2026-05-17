// Package coffeehandlers carries the arabica-app-specific HTTP handlers.
//
// Each handler hangs off *Handlers, which embeds *handlers.Handler so the
// shared infrastructure (auth, atproto store, witness cache, layout data,
// etc.) remains accessible via method promotion.
package coffeehandlers

import (
	"tangled.org/arabica.social/arabica/internal/handlers"
)

// Handlers is the arabica-specific handler set. It embeds the shared
// *handlers.Handler so promoted methods give access to common helpers.
type Handlers struct {
	*handlers.Handler
}

// New constructs a Handlers wrapper over an already-configured base.
// The base handler is shared across all per-app handler sets in a binary.
func New(base *handlers.Handler) *Handlers {
	return &Handlers{Handler: base}
}
