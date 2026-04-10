package atproto

import (
	"maps"
	"sync"
	"time"

	"tangled.org/arabica.social/arabica/internal/models"
)

// CacheTTL is how long cached data remains valid
// Set to 2 minutes to balance multi-device sync with PDS request load
const CacheTTL = 2 * time.Minute

// UserCache holds cached data for a single user.
// This struct is immutable once created - modifications create new instances.
type UserCache struct {
	Beans     []*models.Bean
	Roasters  []*models.Roaster
	Grinders  []*models.Grinder
	Brewers   []*models.Brewer
	Recipes   []*models.Recipe
	Brews     []*models.Brew
	Timestamp time.Time
	// DirtyCollections tracks collections that were recently written to.
	// When a collection is dirty, the witness cache should be skipped
	// because firehose indexing may not have caught up yet.
	DirtyCollections map[string]bool
}

// IsValid returns true if the cache is still valid
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

// clone creates a shallow copy of the UserCache for safe modification
func (c *UserCache) clone() *UserCache {
	if c == nil {
		return &UserCache{Timestamp: time.Now()}
	}
	// Copy dirty collections map
	var dirty map[string]bool
	if c.DirtyCollections != nil {
		dirty = make(map[string]bool, len(c.DirtyCollections))
		maps.Copy(dirty, c.DirtyCollections)
	}
	return &UserCache{
		Beans:            c.Beans,
		Roasters:         c.Roasters,
		Grinders:         c.Grinders,
		Brewers:          c.Brewers,
		Recipes:          c.Recipes,
		Brews:            c.Brews,
		Timestamp:        c.Timestamp,
		DirtyCollections: dirty,
	}
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

// Set stores a user's cache (replaces entirely)
func (sc *SessionCache) Set(sessionID string, cache *UserCache) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.caches[sessionID] = cache
}

// Invalidate removes a user's cache entirely
func (sc *SessionCache) Invalidate(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.caches, sessionID)
}

// SetBeans updates just the beans in the cache using copy-on-write
func (sc *SessionCache) SetBeans(sessionID string, beans []*models.Bean) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	newCache := sc.caches[sessionID].clone()
	newCache.Beans = beans
	newCache.Timestamp = time.Now()
	sc.caches[sessionID] = newCache
}

// SetRoasters updates just the roasters in the cache using copy-on-write
func (sc *SessionCache) SetRoasters(sessionID string, roasters []*models.Roaster) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	newCache := sc.caches[sessionID].clone()
	newCache.Roasters = roasters
	newCache.Timestamp = time.Now()
	sc.caches[sessionID] = newCache
}

// SetGrinders updates just the grinders in the cache using copy-on-write
func (sc *SessionCache) SetGrinders(sessionID string, grinders []*models.Grinder) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	newCache := sc.caches[sessionID].clone()
	newCache.Grinders = grinders
	newCache.Timestamp = time.Now()
	sc.caches[sessionID] = newCache
}

// SetBrewers updates just the brewers in the cache using copy-on-write
func (sc *SessionCache) SetBrewers(sessionID string, brewers []*models.Brewer) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	newCache := sc.caches[sessionID].clone()
	newCache.Brewers = brewers
	newCache.Timestamp = time.Now()
	sc.caches[sessionID] = newCache
}

// SetRecipes updates just the recipes in the cache using copy-on-write
func (sc *SessionCache) SetRecipes(sessionID string, recipes []*models.Recipe) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	newCache := sc.caches[sessionID].clone()
	newCache.Recipes = recipes
	newCache.Timestamp = time.Now()
	sc.caches[sessionID] = newCache
}

// SetBrews updates just the brews in the cache using copy-on-write
func (sc *SessionCache) SetBrews(sessionID string, brews []*models.Brew) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	newCache := sc.caches[sessionID].clone()
	newCache.Brews = brews
	newCache.Timestamp = time.Now()
	sc.caches[sessionID] = newCache
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

// InvalidateBeans marks that beans need to be refreshed using copy-on-write
func (sc *SessionCache) InvalidateBeans(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		newCache := cache.clone()
		newCache.Beans = nil
		markDirty(newCache, NSIDBean)
		sc.caches[sessionID] = newCache
	}
}

// InvalidateRoasters marks that roasters need to be refreshed using copy-on-write
func (sc *SessionCache) InvalidateRoasters(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		newCache := cache.clone()
		newCache.Roasters = nil
		// Also invalidate beans since they reference roasters
		newCache.Beans = nil
		markDirty(newCache, NSIDRoaster)
		sc.caches[sessionID] = newCache
	}
}

// InvalidateGrinders marks that grinders need to be refreshed using copy-on-write
func (sc *SessionCache) InvalidateGrinders(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		newCache := cache.clone()
		newCache.Grinders = nil
		markDirty(newCache, NSIDGrinder)
		sc.caches[sessionID] = newCache
	}
}

// InvalidateBrewers marks that brewers need to be refreshed using copy-on-write
func (sc *SessionCache) InvalidateBrewers(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		newCache := cache.clone()
		newCache.Brewers = nil
		markDirty(newCache, NSIDBrewer)
		sc.caches[sessionID] = newCache
	}
}

// InvalidateRecipes marks that recipes need to be refreshed using copy-on-write
func (sc *SessionCache) InvalidateRecipes(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		newCache := cache.clone()
		newCache.Recipes = nil
		markDirty(newCache, NSIDRecipe)
		sc.caches[sessionID] = newCache
	}
}

// InvalidateBrews marks that brews need to be refreshed using copy-on-write
func (sc *SessionCache) InvalidateBrews(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		newCache := cache.clone()
		newCache.Brews = nil
		markDirty(newCache, NSIDBrew)
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
