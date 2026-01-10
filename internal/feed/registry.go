package feed

import (
	"sync"
)

// Registry tracks known Arabica users (DIDs) for the social feed.
// This is an in-memory store that gets populated as users log in.
// In the future, this could be persisted to a database.
type Registry struct {
	mu   sync.RWMutex
	dids map[string]struct{}
}

// NewRegistry creates a new user registry
func NewRegistry() *Registry {
	return &Registry{
		dids: make(map[string]struct{}),
	}
}

// Register adds a DID to the registry
func (r *Registry) Register(did string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dids[did] = struct{}{}
}

// Unregister removes a DID from the registry
func (r *Registry) Unregister(did string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.dids, did)
}

// IsRegistered checks if a DID is in the registry
func (r *Registry) IsRegistered(did string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.dids[did]
	return ok
}

// List returns all registered DIDs
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dids := make([]string, 0, len(r.dids))
	for did := range r.dids {
		dids = append(dids, did)
	}
	return dids
}

// Count returns the number of registered users
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.dids)
}
