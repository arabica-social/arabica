package atproto

import (
	"arabica/internal/models"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========== UserCache Tests ==========

func TestUserCache_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		cache     *UserCache
		wantValid bool
	}{
		{
			name:      "nil cache is invalid",
			cache:     nil,
			wantValid: false,
		},
		{
			name: "fresh cache is valid",
			cache: &UserCache{
				Timestamp: time.Now(),
			},
			wantValid: true,
		},
		{
			name: "cache within TTL is valid",
			cache: &UserCache{
				Timestamp: time.Now().Add(-CacheTTL / 2),
			},
			wantValid: true,
		},
		{
			name: "cache at TTL boundary is valid",
			cache: &UserCache{
				Timestamp: time.Now().Add(-CacheTTL + time.Millisecond),
			},
			wantValid: true,
		},
		{
			name: "expired cache is invalid",
			cache: &UserCache{
				Timestamp: time.Now().Add(-CacheTTL - time.Second),
			},
			wantValid: false,
		},
		{
			name: "very old cache is invalid",
			cache: &UserCache{
				Timestamp: time.Now().Add(-24 * time.Hour),
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cache.IsValid()
			assert.Equal(t, tt.wantValid, result)
		})
	}
}

func TestUserCache_clone(t *testing.T) {
	t.Run("clone nil cache creates new cache", func(t *testing.T) {
		var cache *UserCache
		cloned := cache.clone()
		require.NotNil(t, cloned)
		assert.NotZero(t, cloned.Timestamp)
	})

	t.Run("clone creates shallow copy", func(t *testing.T) {
		original := &UserCache{
			Beans: []*models.Bean{
				{RKey: "bean1", Name: "Bean One"},
				{RKey: "bean2", Name: "Bean Two"},
			},
			Roasters: []*models.Roaster{
				{RKey: "roaster1", Name: "Roaster One"},
			},
			Grinders: []*models.Grinder{
				{RKey: "grinder1", Name: "Grinder One"},
			},
			Brewers: []*models.Brewer{
				{RKey: "brewer1", Name: "Brewer One"},
			},
			Brews: []*models.Brew{
				{RKey: "brew1", Method: "V60"},
			},
			Timestamp: time.Now(),
		}

		cloned := original.clone()
		require.NotNil(t, cloned)

		// Verify all slices are copied (shallow copy)
		assert.Equal(t, len(original.Beans), len(cloned.Beans))
		assert.Equal(t, len(original.Roasters), len(cloned.Roasters))
		assert.Equal(t, len(original.Grinders), len(cloned.Grinders))
		assert.Equal(t, len(original.Brewers), len(cloned.Brewers))
		assert.Equal(t, len(original.Brews), len(cloned.Brews))
		assert.Equal(t, original.Timestamp, cloned.Timestamp)

		// Verify shallow copy: modifying slice affects both
		original.Beans[0].Name = "Modified"
		assert.Equal(t, "Modified", cloned.Beans[0].Name)
	})

	t.Run("clone is independent reference", func(t *testing.T) {
		original := &UserCache{
			Beans:     []*models.Bean{{RKey: "bean1"}},
			Timestamp: time.Now(),
		}

		cloned := original.clone()

		// Modify original slice reference (not elements)
		original.Beans = []*models.Bean{{RKey: "bean2"}}

		// Cloned should still have old reference
		assert.Equal(t, "bean1", cloned.Beans[0].RKey)
	})
}

// ========== SessionCache Tests ==========

func TestNewSessionCache(t *testing.T) {
	cache := NewSessionCache()
	require.NotNil(t, cache)
	require.NotNil(t, cache.caches)
	assert.Empty(t, cache.caches)
}

func TestSessionCache_GetSetInvalidate(t *testing.T) {
	cache := NewSessionCache()
	sessionID := "session123"

	t.Run("get nonexistent session returns nil", func(t *testing.T) {
		result := cache.Get(sessionID)
		assert.Nil(t, result)
	})

	t.Run("set and get session", func(t *testing.T) {
		userCache := &UserCache{
			Beans:     []*models.Bean{{RKey: "bean1"}},
			Timestamp: time.Now(),
		}

		cache.Set(sessionID, userCache)
		result := cache.Get(sessionID)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(result.Beans))
		assert.Equal(t, "bean1", result.Beans[0].RKey)
	})

	t.Run("invalidate removes session", func(t *testing.T) {
		cache.Invalidate(sessionID)
		result := cache.Get(sessionID)
		assert.Nil(t, result)
	})

	t.Run("invalidate nonexistent session is safe", func(t *testing.T) {
		cache.Invalidate("nonexistent")
		// Should not panic
	})
}

func TestSessionCache_SetCollections(t *testing.T) {
	cache := NewSessionCache()
	sessionID := "session123"

	// Initialize cache with some data
	initial := &UserCache{
		Beans:     []*models.Bean{{RKey: "bean1"}},
		Roasters:  []*models.Roaster{{RKey: "roaster1"}},
		Grinders:  []*models.Grinder{{RKey: "grinder1"}},
		Brewers:   []*models.Brewer{{RKey: "brewer1"}},
		Brews:     []*models.Brew{{RKey: "brew1"}},
		Timestamp: time.Now().Add(-time.Minute),
	}
	cache.Set(sessionID, initial)

	t.Run("SetBeans updates only beans", func(t *testing.T) {
		newBeans := []*models.Bean{
			{RKey: "bean2", Name: "New Bean"},
			{RKey: "bean3", Name: "Another Bean"},
		}

		cache.SetBeans(sessionID, newBeans)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		// Beans should be updated
		assert.Len(t, result.Beans, 2)
		assert.Equal(t, "bean2", result.Beans[0].RKey)

		// Other collections unchanged
		assert.Len(t, result.Roasters, 1)
		assert.Len(t, result.Grinders, 1)
		assert.Len(t, result.Brewers, 1)
		assert.Len(t, result.Brews, 1)

		// Timestamp should be updated
		assert.True(t, result.Timestamp.After(initial.Timestamp))
	})

	t.Run("SetRoasters updates only roasters", func(t *testing.T) {
		newRoasters := []*models.Roaster{{RKey: "roaster2"}}
		cache.SetRoasters(sessionID, newRoasters)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Len(t, result.Roasters, 1)
		assert.Equal(t, "roaster2", result.Roasters[0].RKey)
		assert.Len(t, result.Beans, 2) // From previous test
	})

	t.Run("SetGrinders updates only grinders", func(t *testing.T) {
		newGrinders := []*models.Grinder{{RKey: "grinder2"}}
		cache.SetGrinders(sessionID, newGrinders)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Len(t, result.Grinders, 1)
		assert.Equal(t, "grinder2", result.Grinders[0].RKey)
	})

	t.Run("SetBrewers updates only brewers", func(t *testing.T) {
		newBrewers := []*models.Brewer{{RKey: "brewer2"}}
		cache.SetBrewers(sessionID, newBrewers)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Len(t, result.Brewers, 1)
		assert.Equal(t, "brewer2", result.Brewers[0].RKey)
	})

	t.Run("SetBrews updates only brews", func(t *testing.T) {
		newBrews := []*models.Brew{{RKey: "brew2"}}
		cache.SetBrews(sessionID, newBrews)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Len(t, result.Brews, 1)
		assert.Equal(t, "brew2", result.Brews[0].RKey)
	})
}

func TestSessionCache_InvalidateCollections(t *testing.T) {
	cache := NewSessionCache()
	sessionID := "session123"

	// Initialize cache with all collections
	initial := &UserCache{
		Beans:     []*models.Bean{{RKey: "bean1"}},
		Roasters:  []*models.Roaster{{RKey: "roaster1"}},
		Grinders:  []*models.Grinder{{RKey: "grinder1"}},
		Brewers:   []*models.Brewer{{RKey: "brewer1"}},
		Brews:     []*models.Brew{{RKey: "brew1"}},
		Timestamp: time.Now(),
	}
	cache.Set(sessionID, initial)

	t.Run("InvalidateBeans clears only beans", func(t *testing.T) {
		cache.InvalidateBeans(sessionID)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Nil(t, result.Beans)
		assert.NotNil(t, result.Roasters)
		assert.NotNil(t, result.Grinders)
		assert.NotNil(t, result.Brewers)
		assert.NotNil(t, result.Brews)
	})

	t.Run("InvalidateRoasters clears roasters AND beans", func(t *testing.T) {
		// Reset cache
		cache.Set(sessionID, initial)

		cache.InvalidateRoasters(sessionID)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		// Both roasters and beans should be nil (cascading invalidation)
		assert.Nil(t, result.Roasters)
		assert.Nil(t, result.Beans)
		assert.NotNil(t, result.Grinders)
		assert.NotNil(t, result.Brewers)
		assert.NotNil(t, result.Brews)
	})

	t.Run("InvalidateGrinders clears only grinders", func(t *testing.T) {
		cache.Set(sessionID, initial)

		cache.InvalidateGrinders(sessionID)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Nil(t, result.Grinders)
		assert.NotNil(t, result.Beans)
		assert.NotNil(t, result.Roasters)
	})

	t.Run("InvalidateBrewers clears only brewers", func(t *testing.T) {
		cache.Set(sessionID, initial)

		cache.InvalidateBrewers(sessionID)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Nil(t, result.Brewers)
		assert.NotNil(t, result.Beans)
	})

	t.Run("InvalidateBrews clears only brews", func(t *testing.T) {
		cache.Set(sessionID, initial)

		cache.InvalidateBrews(sessionID)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Nil(t, result.Brews)
		assert.NotNil(t, result.Beans)
	})

	t.Run("invalidate on nonexistent session is safe", func(t *testing.T) {
		cache.InvalidateBeans("nonexistent")
		cache.InvalidateRoasters("nonexistent")
		cache.InvalidateGrinders("nonexistent")
		cache.InvalidateBrewers("nonexistent")
		cache.InvalidateBrews("nonexistent")
		// Should not panic
	})
}

func TestSessionCache_Cleanup(t *testing.T) {
	cache := NewSessionCache()

	// Add fresh cache
	freshCache := &UserCache{
		Beans:     []*models.Bean{{RKey: "bean1"}},
		Timestamp: time.Now(),
	}
	cache.Set("session-fresh", freshCache)

	// Add old cache (beyond 2x TTL)
	oldCache := &UserCache{
		Beans:     []*models.Bean{{RKey: "bean2"}},
		Timestamp: time.Now().Add(-CacheTTL*2 - time.Second),
	}
	cache.Set("session-old", oldCache)

	// Add cache within TTL
	recentCache := &UserCache{
		Beans:     []*models.Bean{{RKey: "bean3"}},
		Timestamp: time.Now().Add(-CacheTTL + time.Minute),
	}
	cache.Set("session-recent", recentCache)

	// Run cleanup
	cache.Cleanup()

	// Fresh and recent should remain
	assert.NotNil(t, cache.Get("session-fresh"))
	assert.NotNil(t, cache.Get("session-recent"))

	// Old should be removed
	assert.Nil(t, cache.Get("session-old"))
}

func TestSessionCache_StartCleanupRoutine(t *testing.T) {
	cache := NewSessionCache()

	// Add old cache
	oldCache := &UserCache{
		Beans:     []*models.Bean{{RKey: "bean1"}},
		Timestamp: time.Now().Add(-CacheTTL*2 - time.Second),
	}
	cache.Set("session-old", oldCache)

	// Start cleanup with very short interval
	stop := cache.StartCleanupRoutine(10 * time.Millisecond)

	// Wait for cleanup to run
	time.Sleep(50 * time.Millisecond)

	// Old cache should be cleaned up
	assert.Nil(t, cache.Get("session-old"))

	// Add another old cache
	cache.Set("session-old2", oldCache)

	// Wait for another cleanup cycle
	time.Sleep(50 * time.Millisecond)

	// Should be cleaned again
	assert.Nil(t, cache.Get("session-old2"))

	// Stop the routine
	stop()

	// Add old cache again
	cache.Set("session-old3", oldCache)

	// Wait - cleanup should not run after stop
	time.Sleep(50 * time.Millisecond)

	// Cache should still exist (cleanup stopped)
	assert.NotNil(t, cache.Get("session-old3"))
}

// ========== Concurrency Tests ==========

func TestSessionCache_ConcurrentAccess(t *testing.T) {
	cache := NewSessionCache()
	numGoroutines := 50
	numOperations := 100

	t.Run("concurrent Set and Get", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(numGoroutines * 2)

		// Writers
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					sessionID := "session"
					userCache := &UserCache{
						Beans:     []*models.Bean{{RKey: "bean"}},
						Timestamp: time.Now(),
					}
					cache.Set(sessionID, userCache)
				}
			}(i)
		}

		// Readers
		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					cache.Get("session")
				}
			}(i)
		}

		wg.Wait()
		// Should not panic or race
	})

	t.Run("concurrent collection updates", func(t *testing.T) {
		sessionID := "test-session"
		initial := &UserCache{
			Beans:     []*models.Bean{{RKey: "bean1"}},
			Roasters:  []*models.Roaster{{RKey: "roaster1"}},
			Timestamp: time.Now(),
		}
		cache.Set(sessionID, initial)

		var wg sync.WaitGroup
		wg.Add(5)

		go func() {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				cache.SetBeans(sessionID, []*models.Bean{{RKey: "bean"}})
			}
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				cache.SetRoasters(sessionID, []*models.Roaster{{RKey: "roaster"}})
			}
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				cache.InvalidateBeans(sessionID)
			}
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				cache.InvalidateRoasters(sessionID)
			}
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				cache.Get(sessionID)
			}
		}()

		wg.Wait()
		// Should not panic or race
	})

	t.Run("concurrent cleanup and access", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(3)

		// Writer
		go func() {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				cache.Set("session", &UserCache{
					Beans:     []*models.Bean{{RKey: "bean"}},
					Timestamp: time.Now(),
				})
			}
		}()

		// Reader
		go func() {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				cache.Get("session")
			}
		}()

		// Cleanup
		go func() {
			defer wg.Done()
			for i := 0; i < numOperations; i++ {
				cache.Cleanup()
			}
		}()

		wg.Wait()
		// Should not panic or race
	})
}

func TestSessionCache_CopyOnWrite(t *testing.T) {
	cache := NewSessionCache()
	sessionID := "session123"

	// Initialize cache
	original := &UserCache{
		Beans:     []*models.Bean{{RKey: "bean1", Name: "Original"}},
		Timestamp: time.Now(),
	}
	cache.Set(sessionID, original)

	// Get reference before update
	before := cache.Get(sessionID)
	require.NotNil(t, before)
	assert.Equal(t, "Original", before.Beans[0].Name)

	// Update beans
	newBeans := []*models.Bean{{RKey: "bean2", Name: "Updated"}}
	cache.SetBeans(sessionID, newBeans)

	// Get reference after update
	after := cache.Get(sessionID)
	require.NotNil(t, after)

	// Verify copy-on-write: old reference still has old data
	assert.Equal(t, "Original", before.Beans[0].Name)
	assert.Equal(t, "Updated", after.Beans[0].Name)

	// Verify they are different instances
	assert.NotEqual(t, before, after)
}

func TestSessionCache_MultipleSessionsIsolation(t *testing.T) {
	cache := NewSessionCache()

	// Create caches for different sessions
	cache.Set("session1", &UserCache{
		Beans:     []*models.Bean{{RKey: "bean1"}},
		Timestamp: time.Now(),
	})

	cache.Set("session2", &UserCache{
		Beans:     []*models.Bean{{RKey: "bean2"}},
		Timestamp: time.Now(),
	})

	cache.Set("session3", &UserCache{
		Beans:     []*models.Bean{{RKey: "bean3"}},
		Timestamp: time.Now(),
	})

	// Update session2
	cache.SetBeans("session2", []*models.Bean{{RKey: "bean2-updated"}})

	// Invalidate session3
	cache.Invalidate("session3")

	// Verify isolation
	s1 := cache.Get("session1")
	require.NotNil(t, s1)
	assert.Equal(t, "bean1", s1.Beans[0].RKey)

	s2 := cache.Get("session2")
	require.NotNil(t, s2)
	assert.Equal(t, "bean2-updated", s2.Beans[0].RKey)

	s3 := cache.Get("session3")
	assert.Nil(t, s3)
}
