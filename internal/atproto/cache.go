package atproto

import (
	"sync"
	"time"

	"arabica/internal/models"
)

// CacheTTL is how long cached data remains valid
const CacheTTL = 5 * time.Minute

// UserCache holds cached data for a single user
type UserCache struct {
	Beans     []*models.Bean
	Roasters  []*models.Roaster
	Grinders  []*models.Grinder
	Brewers   []*models.Brewer
	Brews     []*models.Brew
	Timestamp time.Time
}

// IsValid returns true if the cache is still valid
func (c *UserCache) IsValid() bool {
	if c == nil {
		return false
	}
	return time.Since(c.Timestamp) < CacheTTL
}

// SessionCache manages per-user caches
type SessionCache struct {
	mu     sync.RWMutex
	caches map[string]*UserCache // keyed by session ID
}

// Global session cache instance
var globalCache = &SessionCache{
	caches: make(map[string]*UserCache),
}

// GetSessionCache returns the global session cache
func GetSessionCache() *SessionCache {
	return globalCache
}

// Get retrieves a user's cache by session ID
func (sc *SessionCache) Get(sessionID string) *UserCache {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return sc.caches[sessionID]
}

// Set stores a user's cache
func (sc *SessionCache) Set(sessionID string, cache *UserCache) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.caches[sessionID] = cache
}

// Invalidate removes a user's cache
func (sc *SessionCache) Invalidate(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	delete(sc.caches, sessionID)
}

// InvalidateBeans marks that beans need to be refreshed
func (sc *SessionCache) InvalidateBeans(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		cache.Beans = nil
	}
}

// InvalidateRoasters marks that roasters need to be refreshed
func (sc *SessionCache) InvalidateRoasters(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		cache.Roasters = nil
		// Also invalidate beans since they reference roasters
		cache.Beans = nil
	}
}

// InvalidateGrinders marks that grinders need to be refreshed
func (sc *SessionCache) InvalidateGrinders(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		cache.Grinders = nil
	}
}

// InvalidateBrewers marks that brewers need to be refreshed
func (sc *SessionCache) InvalidateBrewers(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		cache.Brewers = nil
	}
}

// InvalidateBrews marks that brews need to be refreshed
func (sc *SessionCache) InvalidateBrews(sessionID string) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if cache, ok := sc.caches[sessionID]; ok {
		cache.Brews = nil
	}
}

// Cleanup removes expired caches (call periodically)
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
