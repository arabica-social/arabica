package boltstore

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestJoinStore(t *testing.T) *JoinStore {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(Options{Path: dbPath})
	require.NoError(t, err)

	t.Cleanup(func() {
		store.Close()
	})

	return store.JoinStore()
}

func TestJoinStore_SaveAndGet(t *testing.T) {
	store := setupTestJoinStore(t)

	req := &JoinRequest{
		ID:        "join-001",
		Email:     "user@example.com",
		Message:   "I love coffee!",
		CreatedAt: time.Now().Truncate(time.Millisecond),
		IP:        "203.0.113.50",
	}

	err := store.SaveRequest(req)
	require.NoError(t, err)

	retrieved, err := store.GetRequest("join-001")
	require.NoError(t, err)
	require.NotNil(t, retrieved)

	assert.Equal(t, req.ID, retrieved.ID)
	assert.Equal(t, req.Email, retrieved.Email)
	assert.Equal(t, req.Message, retrieved.Message)
	assert.Equal(t, req.IP, retrieved.IP)
	assert.True(t, req.CreatedAt.Equal(retrieved.CreatedAt))
}

func TestJoinStore_GetNotFound(t *testing.T) {
	store := setupTestJoinStore(t)

	retrieved, err := store.GetRequest("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, retrieved)
	assert.Contains(t, err.Error(), "not found")
}

func TestJoinStore_Delete(t *testing.T) {
	store := setupTestJoinStore(t)

	req := &JoinRequest{
		ID:        "join-del",
		Email:     "delete@example.com",
		CreatedAt: time.Now(),
		IP:        "10.0.0.1",
	}

	err := store.SaveRequest(req)
	require.NoError(t, err)

	err = store.DeleteRequest("join-del")
	require.NoError(t, err)

	retrieved, err := store.GetRequest("join-del")
	assert.Error(t, err)
	assert.Nil(t, retrieved)
}

func TestJoinStore_DeleteNonexistent(t *testing.T) {
	store := setupTestJoinStore(t)

	// Deleting a non-existent request should not error
	err := store.DeleteRequest("nonexistent")
	assert.NoError(t, err)
}

func TestJoinStore_ListRequests(t *testing.T) {
	store := setupTestJoinStore(t)

	t.Run("empty store", func(t *testing.T) {
		requests, err := store.ListRequests()
		require.NoError(t, err)
		assert.Empty(t, requests)
	})

	t.Run("multiple requests", func(t *testing.T) {
		for i, email := range []string{"a@test.com", "b@test.com", "c@test.com"} {
			req := &JoinRequest{
				ID:        "list-" + string(rune('0'+i)),
				Email:     email,
				CreatedAt: time.Now(),
				IP:        "10.0.0.1",
			}
			require.NoError(t, store.SaveRequest(req))
		}

		requests, err := store.ListRequests()
		require.NoError(t, err)
		assert.Len(t, requests, 3)
	})
}

func TestJoinStore_SaveOverwrites(t *testing.T) {
	store := setupTestJoinStore(t)

	req := &JoinRequest{
		ID:        "join-overwrite",
		Email:     "original@example.com",
		CreatedAt: time.Now(),
		IP:        "10.0.0.1",
	}

	err := store.SaveRequest(req)
	require.NoError(t, err)

	// Save again with updated email
	req.Email = "updated@example.com"
	err = store.SaveRequest(req)
	require.NoError(t, err)

	retrieved, err := store.GetRequest("join-overwrite")
	require.NoError(t, err)
	assert.Equal(t, "updated@example.com", retrieved.Email)
}
