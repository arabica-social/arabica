package boltstore

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestFeedStore(t *testing.T) *FeedStore {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	store, err := Open(Options{Path: dbPath})
	require.NoError(t, err)

	t.Cleanup(func() {
		store.Close()
	})

	return store.FeedStore()
}

func TestFeedStore_Register(t *testing.T) {
	store := setupTestFeedStore(t)

	t.Run("register new DID", func(t *testing.T) {
		err := store.Register("did:plc:user1")
		require.NoError(t, err)
		assert.True(t, store.IsRegistered("did:plc:user1"))
	})

	t.Run("register is idempotent", func(t *testing.T) {
		err := store.Register("did:plc:user2")
		require.NoError(t, err)

		err = store.Register("did:plc:user2")
		require.NoError(t, err)

		assert.Equal(t, 1, countDID(store, "did:plc:user2"))
	})
}

// countDID counts how many times a DID appears in the list (should be 0 or 1).
func countDID(store *FeedStore, did string) int {
	count := 0
	for _, d := range store.List() {
		if d == did {
			count++
		}
	}
	return count
}

func TestFeedStore_Unregister(t *testing.T) {
	store := setupTestFeedStore(t)

	err := store.Register("did:plc:unreg")
	require.NoError(t, err)
	assert.True(t, store.IsRegistered("did:plc:unreg"))

	err = store.Unregister("did:plc:unreg")
	require.NoError(t, err)
	assert.False(t, store.IsRegistered("did:plc:unreg"))
}

func TestFeedStore_IsRegistered(t *testing.T) {
	store := setupTestFeedStore(t)

	assert.False(t, store.IsRegistered("did:plc:nobody"))

	store.Register("did:plc:somebody")
	assert.True(t, store.IsRegistered("did:plc:somebody"))
}

func TestFeedStore_List(t *testing.T) {
	store := setupTestFeedStore(t)

	t.Run("empty store", func(t *testing.T) {
		dids := store.List()
		assert.Empty(t, dids)
	})

	t.Run("multiple registrations", func(t *testing.T) {
		store.Register("did:plc:a")
		store.Register("did:plc:b")
		store.Register("did:plc:c")

		dids := store.List()
		assert.Len(t, dids, 3)
		assert.Contains(t, dids, "did:plc:a")
		assert.Contains(t, dids, "did:plc:b")
		assert.Contains(t, dids, "did:plc:c")
	})
}

func TestFeedStore_ListWithMetadata(t *testing.T) {
	store := setupTestFeedStore(t)

	store.Register("did:plc:meta1")
	store.Register("did:plc:meta2")

	users := store.ListWithMetadata()
	assert.Len(t, users, 2)

	for _, u := range users {
		assert.NotEmpty(t, u.DID)
		assert.False(t, u.RegisteredAt.IsZero())
	}
}

func TestFeedStore_Count(t *testing.T) {
	store := setupTestFeedStore(t)

	assert.Equal(t, 0, store.Count())

	store.Register("did:plc:c1")
	assert.Equal(t, 1, store.Count())

	store.Register("did:plc:c2")
	assert.Equal(t, 2, store.Count())

	store.Unregister("did:plc:c1")
	assert.Equal(t, 1, store.Count())
}

func TestFeedStore_Clear(t *testing.T) {
	store := setupTestFeedStore(t)

	store.Register("did:plc:clear1")
	store.Register("did:plc:clear2")
	assert.Equal(t, 2, store.Count())

	err := store.Clear()
	require.NoError(t, err)
	assert.Equal(t, 0, store.Count())
	assert.False(t, store.IsRegistered("did:plc:clear1"))
}
