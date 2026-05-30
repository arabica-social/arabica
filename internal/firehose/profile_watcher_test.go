package firehose

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProfileWatcher_AccountDeleted_PurgesData(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	target := "did:plc:victim"

	bean := []byte(`{"$type":"social.arabica.alpha.bean","name":"Bean","createdAt":"2025-01-01T00:00:00Z"}`)
	assert.NoError(t, idx.UpsertRecord(ctx, target, "social.arabica.alpha.bean", "b1", "cid1", bean, time.Now().Unix()))
	assert.Equal(t, 1, idx.RecordCount())

	pw := &ProfileWatcher{
		index:       idx,
		watchedDIDs: map[string]struct{}{target: {}},
	}

	pw.ProcessEvent(accountEvent(target, "deleted"))

	assert.Equal(t, 0, idx.RecordCount(), "records should be purged on account deletion")

	pw.watchedDIDsMu.RLock()
	_, present := pw.watchedDIDs[target]
	pw.watchedDIDsMu.RUnlock()
	assert.False(t, present, "DID should be unwatched after deletion")
}

func TestProfileWatcher_AccountDeactivated_KeepsData(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	target := "did:plc:victim"

	bean := []byte(`{"$type":"social.arabica.alpha.bean","name":"Bean","createdAt":"2025-01-01T00:00:00Z"}`)
	assert.NoError(t, idx.UpsertRecord(ctx, target, "social.arabica.alpha.bean", "b1", "cid1", bean, time.Now().Unix()))

	pw := &ProfileWatcher{
		index:       idx,
		watchedDIDs: map[string]struct{}{target: {}},
	}

	pw.ProcessEvent(accountEvent(target, "deactivated"))

	assert.Equal(t, 1, idx.RecordCount(), "deactivated accounts are reversible — keep data")

	pw.watchedDIDsMu.RLock()
	_, present := pw.watchedDIDs[target]
	pw.watchedDIDsMu.RUnlock()
	assert.True(t, present, "DID should remain watched on reversible status changes")
}

func TestProfileWatcher_AccountTakendown_PurgesData(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	assert.NoError(t, err)
	defer idx.Close()

	ctx := context.Background()
	target := "did:plc:victim"

	bean := []byte(`{"$type":"social.arabica.alpha.bean","name":"Bean","createdAt":"2025-01-01T00:00:00Z"}`)
	assert.NoError(t, idx.UpsertRecord(ctx, target, "social.arabica.alpha.bean", "b1", "cid1", bean, time.Now().Unix()))

	pw := &ProfileWatcher{
		index:       idx,
		watchedDIDs: map[string]struct{}{target: {}},
	}

	pw.ProcessEvent(accountEvent(target, "takendown"))

	assert.Equal(t, 0, idx.RecordCount(), "takendown accounts should be purged")
}

func accountEvent(did, status string) JetstreamEvent {
	return JetstreamEvent{
		DID:    did,
		TimeUS: 1700000000000000,
		Kind:   "account",
		Account: &JetstreamAccount{
			Active: false,
			DID:    did,
			Seq:    1,
			Status: status,
			Time:   "2025-01-03T00:00:00Z",
		},
	}
}
