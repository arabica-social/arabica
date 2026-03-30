package sqlitestore

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"arabica/internal/moderation"

	"github.com/stretchr/testify/assert"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *ModerationStore {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	assert.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`
		CREATE TABLE moderation_labels (
			id          TEXT PRIMARY KEY,
			entity_type TEXT NOT NULL,
			entity_id   TEXT NOT NULL,
			label       TEXT NOT NULL,
			value       TEXT NOT NULL DEFAULT '',
			created_at  TEXT NOT NULL,
			created_by  TEXT NOT NULL,
			expires_at  TEXT,
			UNIQUE(entity_type, entity_id, label)
		);
		CREATE INDEX idx_modlabels_entity ON moderation_labels(entity_type, entity_id);
		CREATE INDEX idx_modlabels_expires ON moderation_labels(expires_at) WHERE expires_at IS NOT NULL;
	`)
	assert.NoError(t, err)
	return NewModerationStore(db)
}

func TestAddAndGetLabel(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	label := moderation.Label{
		ID:         "tid001",
		EntityType: "user",
		EntityID:   "did:plc:test123",
		Name:       "warned",
		Value:      "",
		CreatedAt:  time.Now(),
		CreatedBy:  "did:plc:moderator",
	}

	assert.NoError(t, store.AddLabel(ctx, label))

	got, err := store.GetLabel(ctx, "user", "did:plc:test123", "warned")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "warned", got.Name)
	assert.Equal(t, "user", got.EntityType)
	assert.Equal(t, "did:plc:test123", got.EntityID)
	assert.Equal(t, "did:plc:moderator", got.CreatedBy)
}

func TestHasLabel(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// No label yet
	has, err := store.HasLabel(ctx, "user", "did:plc:test", "warned")
	assert.NoError(t, err)
	assert.False(t, has)

	// Add label
	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID:         "tid001",
		EntityType: "user",
		EntityID:   "did:plc:test",
		Name:       "warned",
		CreatedAt:  time.Now(),
		CreatedBy:  "automod",
	}))

	has, err = store.HasLabel(ctx, "user", "did:plc:test", "warned")
	assert.NoError(t, err)
	assert.True(t, has)
}

func TestHasLabel_Expired(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	past := time.Now().Add(-1 * time.Hour)
	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID:         "tid001",
		EntityType: "user",
		EntityID:   "did:plc:test",
		Name:       "rate_limited",
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		CreatedBy:  "automod",
		ExpiresAt:  &past,
	}))

	// HasLabel should return false for expired labels
	has, err := store.HasLabel(ctx, "user", "did:plc:test", "rate_limited")
	assert.NoError(t, err)
	assert.False(t, has)
}

func TestRemoveLabel(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID:         "tid001",
		EntityType: "user",
		EntityID:   "did:plc:test",
		Name:       "warned",
		CreatedAt:  time.Now(),
		CreatedBy:  "did:plc:mod",
	}))

	assert.NoError(t, store.RemoveLabel(ctx, "user", "did:plc:test", "warned"))

	got, err := store.GetLabel(ctx, "user", "did:plc:test", "warned")
	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestListLabels(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID: "tid001", EntityType: "user", EntityID: "did:plc:test",
		Name: "warned", CreatedAt: time.Now(), CreatedBy: "mod",
	}))
	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID: "tid002", EntityType: "user", EntityID: "did:plc:test",
		Name: "trusted", CreatedAt: time.Now(), CreatedBy: "mod",
	}))
	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID: "tid003", EntityType: "user", EntityID: "did:plc:other",
		Name: "spam", CreatedAt: time.Now(), CreatedBy: "mod",
	}))

	labels, err := store.ListLabels(ctx, "user", "did:plc:test")
	assert.NoError(t, err)
	assert.Len(t, labels, 2)

	all, err := store.ListAllLabels(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestCleanExpiredLabels(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	// Expired label
	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID: "tid001", EntityType: "user", EntityID: "did:plc:a",
		Name: "rate_limited", CreatedAt: time.Now(), CreatedBy: "automod",
		ExpiresAt: &past,
	}))
	// Active label with future expiry
	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID: "tid002", EntityType: "user", EntityID: "did:plc:b",
		Name: "warned", CreatedAt: time.Now(), CreatedBy: "mod",
		ExpiresAt: &future,
	}))
	// Permanent label
	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID: "tid003", EntityType: "user", EntityID: "did:plc:c",
		Name: "trusted", CreatedAt: time.Now(), CreatedBy: "mod",
	}))

	cleaned, err := store.CleanExpiredLabels(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, cleaned)

	// Only 2 labels should remain
	all, err := store.ListAllLabels(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestAddLabel_Upsert(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Add label
	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID: "tid001", EntityType: "user", EntityID: "did:plc:test",
		Name: "warned", Value: "first", CreatedAt: time.Now(), CreatedBy: "mod1",
	}))

	// Upsert same label with new value
	assert.NoError(t, store.AddLabel(ctx, moderation.Label{
		ID: "tid002", EntityType: "user", EntityID: "did:plc:test",
		Name: "warned", Value: "second", CreatedAt: time.Now(), CreatedBy: "mod2",
	}))

	got, err := store.GetLabel(ctx, "user", "did:plc:test", "warned")
	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "second", got.Value)
	assert.Equal(t, "mod2", got.CreatedBy)

	// Should still be only one label
	labels, err := store.ListLabels(ctx, "user", "did:plc:test")
	assert.NoError(t, err)
	assert.Len(t, labels, 1)
}
