package atproto

import (
	"maps"
	"sync"
	"time"
)

// CacheTTL is how long cached data remains valid.
// Set to 2 minutes to balance multi-device sync with PDS request load.
const CacheTTL = 2 * time.Minute

// UserCache holds cached records for a single user, keyed by NSID.
// Values in Records are typed slices (e.g. []*arabica.Bean); the typed
// accessor methods (Beans(), Roasters(), ...) handle the cast.
//
// This struct is treated as immutable once created — modifications create
// new instances via clone().
type UserCache struct {
	Records   map[string]any
	Timestamp time.Time
	// DirtyCollections tracks collections that were recently written to.
	// When a collection is dirty, the witness cache should be skipped
	// because firehose indexing may not have caught up yet.
	DirtyCollections map[string]bool
}

// IsValid returns true if the cache is still valid.
func (c *UserCache) IsValid() bool {
	if c == nil {
		return false
	}
	return time.Since(c.Timestamp) < CacheTTL
}

// IsDirty returns true if the given collection was recently written to
// and the witness cache should be skipped.
func (c *UserCache) IsDirty(collection string) bool {
	if c == nil || c.DirtyCollections == nil {
		return false
	}
	return c.DirtyCollections[collection]
}

// clone creates a shallow copy of the UserCache for safe modification.
// The Records map is copied but its slice values share backing arrays,
// which is fine because callers never mutate cached slices in place.
func (c *UserCache) clone() *UserCache {
	if c == nil {
		return &UserCache{Timestamp: time.Now()}
	}
	var dirty map[string]bool
	if c.DirtyCollections != nil {
		dirty = make(map[string]bool, len(c.DirtyCollections))
		maps.Copy(dirty, c.DirtyCollections)
	}
	var records map[string]any
	if c.Records != nil {
		records = make(map[string]any, len(c.Records))
		maps.Copy(records, c.Records)
	}
	return &UserCache{
		Records:          records,
		Timestamp:        c.Timestamp,
		DirtyCollections: dirty,
	}
}

// CachedSlice retrieves the cached typed slice for a given NSID, or nil
// when the cache is empty, missing the entry, or holds a value of the
// wrong type. Generic over the element type so callers stay typed: the
// previous per-entity accessor methods (Beans/Roasters/etc.) were
// near-clones of this helper.
func CachedSlice[M any](c *UserCache, nsid string) []*M {
	if c == nil {
		return nil
	}
	v, _ := c.Records[nsid].([]*M)
	return v
}

// SessionCache manages per-user caches with proper synchronization.
// Uses copy-on-write pattern to avoid race conditions when reading
// cache entries while other goroutines are modifying them.
type SessionCache struct {
	mu     sync.RWMutex
	caches map[string]*UserCache // keyed by session ID
}

// NewSessionCache creates a new session cache instance.
// Prefer this over global state for better testability and dependency injection.
func NewSessionCache() *SessionCache {
	return &SessionCache{
		caches: make(map[string]*UserCache),
	}
}

// Get retrieves a user's cache by session ID.
// The returned UserCache is safe to read without holding a lock.
func (sc *SessionCache) Get(sessionID string) *UserCache {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.caches[sessionID]
}

// Set stores a user's cache (replaces entirely).
func (sc *SessionCache) Set(sessionID string, cache *UserCache) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.caches[sessionID] = cache
}

// Invalidate removes a user's cache entirely.
func (sc *SessionCache) Invalidate(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.caches, sessionID)
}

// SetRecords stores records for one NSID using copy-on-write. The records
// argument should be a typed slice (e.g. []*arabica.Bean); SessionCache does
// not interpret the value.
func (sc *SessionCache) SetRecords(sessionID, nsid string, records any) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	newCache := sc.caches[sessionID].clone()
	if newCache.Records == nil {
		newCache.Records = make(map[string]any)
	}
	newCache.Records[nsid] = records
	newCache.Timestamp = time.Now()
	sc.caches[sessionID] = newCache
}

// InvalidateRecords clears the cache for one NSID and marks it dirty so the
// witness cache is skipped until firehose catches up.
func (sc *SessionCache) InvalidateRecords(sessionID, nsid string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		newCache := cache.clone()
		delete(newCache.Records, nsid)
		markDirty(newCache, nsid)
		sc.caches[sessionID] = newCache
	}
}

// markDirty sets a collection as dirty on the given cache, initializing the map if needed.
func markDirty(cache *UserCache, collection string) {
	if cache.DirtyCollections == nil {
		cache.DirtyCollections = make(map[string]bool)
	}
	cache.DirtyCollections[collection] = true
}

// ClearDirty removes the dirty flag for a collection after fresh PDS data has been cached.
func (sc *SessionCache) ClearDirty(sessionID, collection string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		newCache := cache.clone()
		delete(newCache.DirtyCollections, collection)
		sc.caches[sessionID] = newCache
	}
}

// Cleanup removes expired caches.
// This should be called periodically by a background goroutine.
func (sc *SessionCache) Cleanup() {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	now := time.Now()
	for sessionID, cache := range sc.caches {
		if now.Sub(cache.Timestamp) > CacheTTL*2 {
			delete(sc.caches, sessionID)
		}
	}
}

// StartCleanupRoutine starts a background goroutine that periodically cleans up
// expired cache entries. Returns a stop function to gracefully shut down.
func (sc *SessionCache) StartCleanupRoutine(interval time.Duration) (stop func()) {
	ticker := time.NewTicker(interval)
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ticker.C:
				sc.Cleanup()
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()

	return func() {
		close(done)
	}
}
