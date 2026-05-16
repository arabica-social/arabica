package atproto

import (
	"sync"
	"testing"
	"time"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"

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
			Records: map[string]any{
				arabica.NSIDBean: []*arabica.Bean{
					{RKey: "bean1", Name: "Bean One"},
					{RKey: "bean2", Name: "Bean Two"},
				},
				arabica.NSIDRoaster: []*arabica.Roaster{
					{RKey: "roaster1", Name: "Roaster One"},
				},
				arabica.NSIDGrinder: []*arabica.Grinder{
					{RKey: "grinder1", Name: "Grinder One"},
				},
				arabica.NSIDBrewer: []*arabica.Brewer{
					{RKey: "brewer1", Name: "Brewer One"},
				},
				arabica.NSIDBrew: []*arabica.Brew{
					{RKey: "brew1", Method: "V60"},
				},
			},
			Timestamp: time.Now(),
		}

		cloned := original.clone()
		require.NotNil(t, cloned)

		// Verify all slices are copied (shallow copy)
		assert.Equal(t, len(CachedSlice[arabica.Bean](original, arabica.NSIDBean)), len(CachedSlice[arabica.Bean](cloned, arabica.NSIDBean)))
		assert.Equal(t, len(CachedSlice[arabica.Roaster](original, arabica.NSIDRoaster)), len(CachedSlice[arabica.Roaster](cloned, arabica.NSIDRoaster)))
		assert.Equal(t, len(CachedSlice[arabica.Grinder](original, arabica.NSIDGrinder)), len(CachedSlice[arabica.Grinder](cloned, arabica.NSIDGrinder)))
		assert.Equal(t, len(CachedSlice[arabica.Brewer](original, arabica.NSIDBrewer)), len(CachedSlice[arabica.Brewer](cloned, arabica.NSIDBrewer)))
		assert.Equal(t, len(CachedSlice[arabica.Brew](original, arabica.NSIDBrew)), len(CachedSlice[arabica.Brew](cloned, arabica.NSIDBrew)))
		assert.Equal(t, original.Timestamp, cloned.Timestamp)

		// Verify shallow copy: modifying slice element affects both
		CachedSlice[arabica.Bean](original, arabica.NSIDBean)[0].Name = "Modified"
		assert.Equal(t, "Modified", CachedSlice[arabica.Bean](cloned, arabica.NSIDBean)[0].Name)
	})

	t.Run("clone is independent reference", func(t *testing.T) {
		original := &UserCache{
			Records: map[string]any{
				arabica.NSIDBean: []*arabica.Bean{{RKey: "bean1"}},
			},
			Timestamp: time.Now(),
		}

		cloned := original.clone()

		// Replace the slice in the original's map (clone should be independent)
		original.Records[arabica.NSIDBean] = []*arabica.Bean{{RKey: "bean2"}}

		// Cloned should still have old reference
		assert.Equal(t, "bean1", CachedSlice[arabica.Bean](cloned, arabica.NSIDBean)[0].RKey)
	})
}

func TestNewSessionCache(t *testing.T) {
	cache := NewSessionCache()
	require.NotNil(t, cache)
	assert.NotNil(t, cache.caches)
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
			Records: map[string]any{
				arabica.NSIDBean: []*arabica.Bean{{RKey: "bean1"}},
			},
			Timestamp: time.Now(),
		}

		cache.Set(sessionID, userCache)
		result := cache.Get(sessionID)
		require.NotNil(t, result)
		assert.Equal(t, 1, len(CachedSlice[arabica.Bean](result, arabica.NSIDBean)))
		assert.Equal(t, "bean1", CachedSlice[arabica.Bean](result, arabica.NSIDBean)[0].RKey)
	})

	t.Run("invalidate removes session", func(t *testing.T) {
		cache.Invalidate(sessionID)
		result := cache.Get(sessionID)
		assert.Nil(t, result)
	})

	t.Run("invalidate nonexistent session is safe", func(t *testing.T) {
		cache.Invalidate("nonexistent")
	})
}

func TestSessionCache_SetCollections(t *testing.T) {
	cache := NewSessionCache()
	sessionID := "session123"

	// Initialize cache with some data
	initial := &UserCache{
		Records: map[string]any{
			arabica.NSIDBean:    []*arabica.Bean{{RKey: "bean1"}},
			arabica.NSIDRoaster: []*arabica.Roaster{{RKey: "roaster1"}},
			arabica.NSIDGrinder: []*arabica.Grinder{{RKey: "grinder1"}},
			arabica.NSIDBrewer:  []*arabica.Brewer{{RKey: "brewer1"}},
			arabica.NSIDBrew:    []*arabica.Brew{{RKey: "brew1"}},
		},
		Timestamp: time.Now().Add(-time.Minute),
	}
	cache.Set(sessionID, initial)

	t.Run("SetBeans updates only beans", func(t *testing.T) {
		newBeans := []*arabica.Bean{
			{RKey: "bean2", Name: "New Bean"},
			{RKey: "bean3", Name: "Another Bean"},
		}

		cache.SetRecords(sessionID, arabica.NSIDBean, newBeans)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		// Beans should be updated
		assert.Len(t, CachedSlice[arabica.Bean](result, arabica.NSIDBean), 2)
		assert.Equal(t, "bean2", CachedSlice[arabica.Bean](result, arabica.NSIDBean)[0].RKey)

		// Other collections unchanged
		assert.Len(t, CachedSlice[arabica.Roaster](result, arabica.NSIDRoaster), 1)
		assert.Len(t, CachedSlice[arabica.Grinder](result, arabica.NSIDGrinder), 1)
		assert.Len(t, CachedSlice[arabica.Brewer](result, arabica.NSIDBrewer), 1)
		assert.Len(t, CachedSlice[arabica.Brew](result, arabica.NSIDBrew), 1)

		// Timestamp should be updated
		assert.True(t, result.Timestamp.After(initial.Timestamp))
	})

	t.Run("SetRoasters updates only roasters", func(t *testing.T) {
		newRoasters := []*arabica.Roaster{{RKey: "roaster2"}}
		cache.SetRecords(sessionID, arabica.NSIDRoaster, newRoasters)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Len(t, CachedSlice[arabica.Roaster](result, arabica.NSIDRoaster), 1)
		assert.Equal(t, "roaster2", CachedSlice[arabica.Roaster](result, arabica.NSIDRoaster)[0].RKey)
		assert.Len(t, CachedSlice[arabica.Bean](result, arabica.NSIDBean), 2) // From previous test
	})

	t.Run("SetGrinders updates only grinders", func(t *testing.T) {
		newGrinders := []*arabica.Grinder{{RKey: "grinder2"}}
		cache.SetRecords(sessionID, arabica.NSIDGrinder, newGrinders)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Len(t, CachedSlice[arabica.Grinder](result, arabica.NSIDGrinder), 1)
		assert.Equal(t, "grinder2", CachedSlice[arabica.Grinder](result, arabica.NSIDGrinder)[0].RKey)
	})

	t.Run("SetBrewers updates only brewers", func(t *testing.T) {
		newBrewers := []*arabica.Brewer{{RKey: "brewer2"}}
		cache.SetRecords(sessionID, arabica.NSIDBrewer, newBrewers)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Len(t, CachedSlice[arabica.Brewer](result, arabica.NSIDBrewer), 1)
		assert.Equal(t, "brewer2", CachedSlice[arabica.Brewer](result, arabica.NSIDBrewer)[0].RKey)
	})

	t.Run("SetBrews updates only brews", func(t *testing.T) {
		newBrews := []*arabica.Brew{{RKey: "brew2"}}
		cache.SetRecords(sessionID, arabica.NSIDBrew, newBrews)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Len(t, CachedSlice[arabica.Brew](result, arabica.NSIDBrew), 1)
		assert.Equal(t, "brew2", CachedSlice[arabica.Brew](result, arabica.NSIDBrew)[0].RKey)
	})
}

func TestSessionCache_InvalidateCollections(t *testing.T) {
	cache := NewSessionCache()
	sessionID := "session123"

	// freshInitial returns a new initialized cache so each subtest can reset.
	freshInitial := func() *UserCache {
		return &UserCache{
			Records: map[string]any{
				arabica.NSIDBean:    []*arabica.Bean{{RKey: "bean1"}},
				arabica.NSIDRoaster: []*arabica.Roaster{{RKey: "roaster1"}},
				arabica.NSIDGrinder: []*arabica.Grinder{{RKey: "grinder1"}},
				arabica.NSIDBrewer:  []*arabica.Brewer{{RKey: "brewer1"}},
				arabica.NSIDBrew:    []*arabica.Brew{{RKey: "brew1"}},
			},
			Timestamp: time.Now(),
		}
	}

	cache.Set(sessionID, freshInitial())

	t.Run("InvalidateBeans clears only beans", func(t *testing.T) {
		cache.InvalidateRecords(sessionID, arabica.NSIDBean)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Nil(t, CachedSlice[arabica.Bean](result, arabica.NSIDBean))
		assert.NotNil(t, CachedSlice[arabica.Roaster](result, arabica.NSIDRoaster))
		assert.NotNil(t, CachedSlice[arabica.Grinder](result, arabica.NSIDGrinder))
		assert.NotNil(t, CachedSlice[arabica.Brewer](result, arabica.NSIDBrewer))
		assert.NotNil(t, CachedSlice[arabica.Brew](result, arabica.NSIDBrew))
	})

	t.Run("Roaster invalidation cascades to beans (call-site pattern)", func(t *testing.T) {
		// Reset cache. Cross-collection invalidation moved from a typed
		// wrapper (Phase C) to an explicit two-call pattern at the call
		// site (Phase D). UpdateRoasterByRKey/DeleteRoasterByRKey in
		// store.go now make both invalidation calls.
		cache.Set(sessionID, freshInitial())

		cache.InvalidateRecords(sessionID, arabica.NSIDRoaster)
		cache.InvalidateRecords(sessionID, arabica.NSIDBean)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Nil(t, CachedSlice[arabica.Roaster](result, arabica.NSIDRoaster))
		assert.Nil(t, CachedSlice[arabica.Bean](result, arabica.NSIDBean))
		assert.NotNil(t, CachedSlice[arabica.Grinder](result, arabica.NSIDGrinder))
		assert.NotNil(t, CachedSlice[arabica.Brewer](result, arabica.NSIDBrewer))
		assert.NotNil(t, CachedSlice[arabica.Brew](result, arabica.NSIDBrew))
	})

	t.Run("InvalidateGrinders clears only grinders", func(t *testing.T) {
		cache.Set(sessionID, freshInitial())

		cache.InvalidateRecords(sessionID, arabica.NSIDGrinder)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Nil(t, CachedSlice[arabica.Grinder](result, arabica.NSIDGrinder))
		assert.NotNil(t, CachedSlice[arabica.Bean](result, arabica.NSIDBean))
		assert.NotNil(t, CachedSlice[arabica.Roaster](result, arabica.NSIDRoaster))
	})

	t.Run("InvalidateBrewers clears only brewers", func(t *testing.T) {
		cache.Set(sessionID, freshInitial())

		cache.InvalidateRecords(sessionID, arabica.NSIDBrewer)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Nil(t, CachedSlice[arabica.Brewer](result, arabica.NSIDBrewer))
		assert.NotNil(t, CachedSlice[arabica.Bean](result, arabica.NSIDBean))
	})

	t.Run("InvalidateBrews clears only brews", func(t *testing.T) {
		cache.Set(sessionID, freshInitial())

		cache.InvalidateRecords(sessionID, arabica.NSIDBrew)
		result := cache.Get(sessionID)
		require.NotNil(t, result)

		assert.Nil(t, CachedSlice[arabica.Brew](result, arabica.NSIDBrew))
		assert.NotNil(t, CachedSlice[arabica.Bean](result, arabica.NSIDBean))
	})

	t.Run("invalidate on nonexistent session is safe", func(t *testing.T) {
		cache.InvalidateRecords("nonexistent", arabica.NSIDBean)
		cache.InvalidateRecords("nonexistent", arabica.NSIDRoaster)
		cache.InvalidateRecords("nonexistent", arabica.NSIDGrinder)
		cache.InvalidateRecords("nonexistent", arabica.NSIDBrewer)
		cache.InvalidateRecords("nonexistent", arabica.NSIDBrew)
		// Should not panic
	})
}

func TestSessionCache_Cleanup(t *testing.T) {
	cache := NewSessionCache()

	// Add fresh cache
	freshCache := &UserCache{
		Records: map[string]any{
			arabica.NSIDBean: []*arabica.Bean{{RKey: "bean1"}},
		},
		Timestamp: time.Now(),
	}
	cache.Set("session-fresh", freshCache)

	// Add old cache (beyond 2x TTL)
	oldCache := &UserCache{
		Records: map[string]any{
			arabica.NSIDBean: []*arabica.Bean{{RKey: "bean2"}},
		},
		Timestamp: time.Now().Add(-CacheTTL*2 - time.Second),
	}
	cache.Set("session-old", oldCache)

	// Add cache within TTL
	recentCache := &UserCache{
		Records: map[string]any{
			arabica.NSIDBean: []*arabica.Bean{{RKey: "bean3"}},
		},
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
		Records: map[string]any{
			arabica.NSIDBean: []*arabica.Bean{{RKey: "bean1"}},
		},
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
		for i := range numGoroutines {
			go func(id int) {
				defer wg.Done()
				for range numOperations {
					sessionID := "session"
					userCache := &UserCache{
						Records: map[string]any{
							arabica.NSIDBean: []*arabica.Bean{{RKey: "bean"}},
						},
						Timestamp: time.Now(),
					}
					cache.Set(sessionID, userCache)
				}
			}(i)
		}

		// Readers
		for i := range numGoroutines {
			go func(id int) {
				defer wg.Done()
				for range numOperations {
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
			Records: map[string]any{
				arabica.NSIDBean:    []*arabica.Bean{{RKey: "bean1"}},
				arabica.NSIDRoaster: []*arabica.Roaster{{RKey: "roaster1"}},
			},
			Timestamp: time.Now(),
		}
		cache.Set(sessionID, initial)

		var wg sync.WaitGroup
		wg.Add(5)

		go func() {
			defer wg.Done()
			for range numOperations {
				cache.SetRecords(sessionID, arabica.NSIDBean, []*arabica.Bean{{RKey: "bean"}})
			}
		}()

		go func() {
			defer wg.Done()
			for range numOperations {
				cache.SetRecords(sessionID, arabica.NSIDRoaster, []*arabica.Roaster{{RKey: "roaster"}})
			}
		}()

		go func() {
			defer wg.Done()
			for range numOperations {
				cache.InvalidateRecords(sessionID, arabica.NSIDBean)
			}
		}()

		go func() {
			defer wg.Done()
			for range numOperations {
				cache.InvalidateRecords(sessionID, arabica.NSIDRoaster)
				cache.InvalidateRecords(sessionID, arabica.NSIDBean)
			}
		}()

		go func() {
			defer wg.Done()
			for range numOperations {
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
			for range numOperations {
				cache.Set("session", &UserCache{
					Records: map[string]any{
						arabica.NSIDBean: []*arabica.Bean{{RKey: "bean"}},
					},
					Timestamp: time.Now(),
				})
			}
		}()

		// Reader
		go func() {
			defer wg.Done()
			for range numOperations {
				cache.Get("session")
			}
		}()

		// Cleanup
		go func() {
			defer wg.Done()
			for range numOperations {
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
		Records: map[string]any{
			arabica.NSIDBean: []*arabica.Bean{{RKey: "bean1", Name: "Original"}},
		},
		Timestamp: time.Now(),
	}
	cache.Set(sessionID, original)

	// Get reference before update
	before := cache.Get(sessionID)
	require.NotNil(t, before)
	assert.Equal(t, "Original", CachedSlice[arabica.Bean](before, arabica.NSIDBean)[0].Name)

	// Update beans
	newBeans := []*arabica.Bean{{RKey: "bean2", Name: "Updated"}}
	cache.SetRecords(sessionID, arabica.NSIDBean, newBeans)

	// Get reference after update
	after := cache.Get(sessionID)
	require.NotNil(t, after)

	// Verify copy-on-write: old reference still has old data
	assert.Equal(t, "Original", CachedSlice[arabica.Bean](before, arabica.NSIDBean)[0].Name)
	assert.Equal(t, "Updated", CachedSlice[arabica.Bean](after, arabica.NSIDBean)[0].Name)

	// Verify they are different instances
	assert.NotEqual(t, before, after)
}

// TestSessionCache_SetRecordsGenericNSID proves SetRecords/InvalidateRecords
// work for any NSID/type combination, not just arabica entities. This is
// the contract a sister app (oolong) depends on.
func TestSessionCache_SetRecordsGenericNSID(t *testing.T) {
	sc := NewSessionCache()
	sessionID := "test-session"

	const fakeNSID = "social.fake.alpha.widget"
	type widget struct{ ID string }
	widgets := []*widget{{ID: "w1"}, {ID: "w2"}}

	sc.SetRecords(sessionID, fakeNSID, widgets)

	cache := sc.Get(sessionID)
	require.NotNil(t, cache)
	got, ok := cache.Records[fakeNSID].([]*widget)
	assert.True(t, ok)
	assert.Equal(t, widgets, got)

	// Not dirty right after Set
	assert.False(t, cache.IsDirty(fakeNSID))

	sc.InvalidateRecords(sessionID, fakeNSID)
	cache = sc.Get(sessionID)
	_, present := cache.Records[fakeNSID]
	assert.False(t, present)
	assert.True(t, cache.IsDirty(fakeNSID))
}

func TestSessionCache_MultipleSessionsIsolation(t *testing.T) {
	cache := NewSessionCache()

	// Create caches for different sessions
	cache.Set("session1", &UserCache{
		Records: map[string]any{
			arabica.NSIDBean: []*arabica.Bean{{RKey: "bean1"}},
		},
		Timestamp: time.Now(),
	})

	cache.Set("session2", &UserCache{
		Records: map[string]any{
			arabica.NSIDBean: []*arabica.Bean{{RKey: "bean2"}},
		},
		Timestamp: time.Now(),
	})

	cache.Set("session3", &UserCache{
		Records: map[string]any{
			arabica.NSIDBean: []*arabica.Bean{{RKey: "bean3"}},
		},
		Timestamp: time.Now(),
	})

	// Update session2
	cache.SetRecords("session2", arabica.NSIDBean, []*arabica.Bean{{RKey: "bean2-updated"}})

	// Invalidate session3
	cache.Invalidate("session3")

	// Verify isolation
	s1 := cache.Get("session1")
	require.NotNil(t, s1)
	assert.Equal(t, "bean1", CachedSlice[arabica.Bean](s1, arabica.NSIDBean)[0].RKey)

	s2 := cache.Get("session2")
	require.NotNil(t, s2)
	assert.Equal(t, "bean2-updated", CachedSlice[arabica.Bean](s2, arabica.NSIDBean)[0].RKey)

	s3 := cache.Get("session3")
	assert.Nil(t, s3)
}
