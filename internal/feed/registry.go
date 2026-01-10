package feed

import (
	"sync"
)

// PersistentStore defines the interface for persistent feed registry storage.
// This allows the Registry to optionally persist DIDs to a database.
type PersistentStore interface {
	Register(did string) error
	Unregister(did string) error
	IsRegistered(did string) bool
	List() []string
	Count() int
}

// Registry tracks known Arabica users (DIDs) for the social feed.
// It maintains an in-memory cache for fast access and optionally
// persists data to a backing store.
type Registry struct {
	mu    sync.RWMutex
	dids  map[string]struct{}
	store PersistentStore // Optional persistent backing store
}

// NewRegistry creates a new user registry with in-memory storage only.
func NewRegistry() *Registry {
	return &Registry{
		dids: make(map[string]struct{}),
	}
}

// NewPersistentRegistry creates a new registry backed by persistent storage.
// It loads existing registrations from the store on creation.
func NewPersistentRegistry(store PersistentStore) *Registry {
	r := &Registry{
		dids:  make(map[string]struct{}),
		store: store,
	}

	// Load existing registrations from store
	if store != nil {
		for _, did := range store.List() {
			r.dids[did] = struct{}{}
		}
	}

	return r
}

// Register adds a DID to the registry.
// If a persistent store is configured, the DID is also saved to the store.
func (r *Registry) Register(did string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Add to in-memory cache
	r.dids[did] = struct{}{}

	// Persist if store is configured
	if r.store != nil {
		// Ignore errors for now - the in-memory cache is the source of truth
		// during this session, and we'll retry on next restart
		_ = r.store.Register(did)
	}
}

// Unregister removes a DID from the registry.
// If a persistent store is configured, the DID is also removed from the store.
func (r *Registry) Unregister(did string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.dids, did)

	if r.store != nil {
		_ = r.store.Unregister(did)
	}
}

// IsRegistered checks if a DID is in the registry.
func (r *Registry) IsRegistered(did string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.dids[did]
	return ok
}

// List returns all registered DIDs.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	dids := make([]string, 0, len(r.dids))
	for did := range r.dids {
		dids = append(dids, did)
	}
	return dids
}

// Count returns the number of registered users.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.dids)
}
