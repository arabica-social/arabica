package integration

import (
	"context"
	"testing"
	"time"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/firehose"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests cover the bug where two accounts share a handle (because the
// first account's PDS went away and the user re-registered with a new DID)
// and the witness cache could resolve the handle to the orphan DID. The fix
// keys handle resolution off a `did_by_handle` table with last-writer-wins
// semantics, plus identity-event-driven invalidation of stale prior owners.

const sharedHandle = "shared.test"

// seedProfile writes a profile cache entry for did/handle without going
// through the public API. Mirrors what the profile watcher would do after
// observing a profile commit on the firehose.
func seedProfile(t *testing.T, h *Harness, did, handle string) {
	t.Helper()
	h.FeedIndex.StoreProfile(context.Background(), did, &atproto.Profile{
		DID:    did,
		Handle: handle,
	})
}

// TestHandleReassignment_NewOwnerWinsLookup is the direct repro of the bug.
// Two DIDs claim the same handle in sequence; the lookup must return the
// most-recently-written owner.
func TestHandleReassignment_NewOwnerWinsLookup(t *testing.T) {
	h := StartHarness(t, nil)
	ctx := context.Background()

	const orphanDID = "did:plc:orphanaccount"
	const newDID = "did:plc:freshaccount"

	seedProfile(t, h, orphanDID, sharedHandle)

	got, ok := h.FeedIndex.GetDIDByHandle(ctx, sharedHandle)
	require.True(t, ok, "lookup should find the seeded profile")
	assert.Equal(t, orphanDID, got)

	// The new account claims the same handle. After its profile is observed,
	// the handle must resolve to the new DID, not the orphan.
	seedProfile(t, h, newDID, sharedHandle)

	got, ok = h.FeedIndex.GetDIDByHandle(ctx, sharedHandle)
	require.True(t, ok)
	assert.Equal(t, newDID, got, "handle should resolve to the most recent owner, not the orphan")
}

// TestHandleChange_OldHandleStopsResolving covers a single DID changing its
// handle (no reassignment between DIDs). The stale handle row must be cleared
// so the old handle no longer resolves.
func TestHandleChange_OldHandleStopsResolving(t *testing.T) {
	h := StartHarness(t, nil)
	ctx := context.Background()

	const did = "did:plc:movinghandles"

	seedProfile(t, h, did, "old.test")

	got, ok := h.FeedIndex.GetDIDByHandle(ctx, "old.test")
	require.True(t, ok)
	assert.Equal(t, did, got)

	seedProfile(t, h, did, "new.test")

	_, ok = h.FeedIndex.GetDIDByHandle(ctx, "old.test")
	assert.False(t, ok, "old handle should no longer resolve after the DID changes its handle")

	got, ok = h.FeedIndex.GetDIDByHandle(ctx, "new.test")
	require.True(t, ok)
	assert.Equal(t, did, got)
}

// TestPurgeDID_FreesHandleForReuse reproduces the cleanup path: an orphan DID
// is purged, then a new account claiming the same handle resolves cleanly.
// The admin /_mod/purge endpoint isn't reachable from the harness (no
// moderation service), so we exercise the same DeleteAllByDID code path it
// invokes and assert the same downstream invariants.
func TestPurgeDID_FreesHandleForReuse(t *testing.T) {
	h := StartHarness(t, nil)
	ctx := context.Background()

	const orphanDID = "did:plc:orphandead"
	const newDID = "did:plc:newowner"

	seedProfile(t, h, orphanDID, sharedHandle)

	// The orphan also has indexed records — purge must remove them.
	const collection = atproto.NSIDRoaster
	const rkey = "orphanrkey"
	now := time.Now().Unix()
	require.NoError(t, h.FeedIndex.UpsertRecord(
		ctx, orphanDID, collection, rkey, "ciddead",
		[]byte(`{"$type":"social.arabica.alpha.roaster","name":"Orphan Roaster","createdAt":"2025-01-01T00:00:00Z"}`),
		now,
	))

	uri := atproto.BuildATURI(orphanDID, collection, rkey)
	rec, err := h.FeedIndex.GetRecord(ctx, uri)
	require.NoError(t, err)
	require.NotNil(t, rec, "orphan record should be indexed before purge")

	require.NoError(t, h.FeedIndex.DeleteAllByDID(ctx, orphanDID))
	h.FeedIndex.InvalidatePublicCachesForDID(orphanDID)

	rec, err = h.FeedIndex.GetRecord(ctx, uri)
	assert.NoError(t, err)
	assert.Nil(t, rec, "orphan record should be gone after purge")

	_, ok := h.FeedIndex.GetDIDByHandle(ctx, sharedHandle)
	assert.False(t, ok, "purge should remove the handle index entry")

	// New account claims the freed handle.
	seedProfile(t, h, newDID, sharedHandle)
	got, ok := h.FeedIndex.GetDIDByHandle(ctx, sharedHandle)
	require.True(t, ok)
	assert.Equal(t, newDID, got, "freed handle should resolve to the new owner")
}

// TestIdentityEvent_EvictsPriorOwnerOfHandle drives the OnIdentityEvent
// reconciliation path through the profile watcher. When a Jetstream identity
// event reports that a new DID has claimed an existing handle, the prior
// owner's profile cache and handle mapping must be invalidated immediately —
// before the eventual API refresh happens — so a concurrent lookup of the
// shared handle never returns the stale owner.
func TestIdentityEvent_EvictsPriorOwnerOfHandle(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})
	ctx := context.Background()

	const orphanDID = "did:plc:identityorphan"
	const newDID = "did:plc:identitynew"

	seedProfile(t, h, orphanDID, sharedHandle)
	got, ok := h.FeedIndex.GetDIDByHandle(ctx, sharedHandle)
	require.True(t, ok)
	require.Equal(t, orphanDID, got, "precondition: orphan owns the handle")

	// Synthesize an identity event for the new DID claiming the shared handle.
	// OnIdentityEvent runs prior-owner eviction synchronously; the subsequent
	// RefreshProfile call hits the public bsky API and fails for our synthetic
	// DID, which is fine — we're verifying the eviction half of the path.
	require.NotNil(t, h.ProfileWatcher)
	h.ProfileWatcher.ProcessEvent(firehose.JetstreamEvent{
		DID:    newDID,
		TimeUS: time.Now().UnixMicro(),
		Kind:   "identity",
		Identity: &firehose.JetstreamIdentity{
			DID:    newDID,
			Handle: sharedHandle,
			Time:   time.Now().Format(time.RFC3339),
		},
	})

	// The orphan must no longer be the owner of the shared handle. Either the
	// lookup misses entirely (because RefreshProfile failed and InvalidateProfile
	// cleared everything) or it resolves to newDID — both are acceptable; what
	// must not happen is the stale orphan winning.
	got, ok = h.FeedIndex.GetDIDByHandle(ctx, sharedHandle)
	if ok {
		assert.NotEqual(t, orphanDID, got, "shared handle must not resolve to the evicted prior owner")
	}

	// Now seed the new DID's profile (as the watcher would, on a working API).
	// The handle must resolve to newDID, never orphanDID.
	seedProfile(t, h, newDID, sharedHandle)
	got, ok = h.FeedIndex.GetDIDByHandle(ctx, sharedHandle)
	require.True(t, ok)
	assert.Equal(t, newDID, got, "after reconciliation, shared handle resolves to the new owner")
}

// TestHandleResolution_BackfillFromExistingProfiles verifies that a FeedIndex
// opened against a database with pre-existing profile rows but no
// did_by_handle table populates the index on startup, so handle lookups work
// for users observed before this fix shipped.
func TestHandleResolution_BackfillFromExistingProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/feed-backfill.db"

	// First open: seed two profiles, then close.
	idx, err := firehose.NewFeedIndex(dbPath, time.Hour)
	require.NoError(t, err)
	idx.StoreProfile(context.Background(), "did:plc:backfilla", &atproto.Profile{
		DID: "did:plc:backfilla", Handle: "alice.bf",
	})
	idx.StoreProfile(context.Background(), "did:plc:backfillb", &atproto.Profile{
		DID: "did:plc:backfillb", Handle: "bob.bf",
	})
	require.NoError(t, idx.Close())

	// Drop did_by_handle to simulate a database that predates the index.
	idx2, err := firehose.NewFeedIndex(dbPath, time.Hour)
	require.NoError(t, err)
	defer idx2.Close()

	_, err = idx2.DB().Exec(`DELETE FROM did_by_handle`)
	require.NoError(t, err)
	require.NoError(t, idx2.Close())

	// Reopen — backfill should re-populate the table from the profile rows.
	idx3, err := firehose.NewFeedIndex(dbPath, time.Hour)
	require.NoError(t, err)
	defer idx3.Close()

	got, ok := idx3.GetDIDByHandle(context.Background(), "alice.bf")
	require.True(t, ok, "backfill should restore alice's handle mapping")
	assert.Equal(t, "did:plc:backfilla", got)

	got, ok = idx3.GetDIDByHandle(context.Background(), "bob.bf")
	require.True(t, ok, "backfill should restore bob's handle mapping")
	assert.Equal(t, "did:plc:backfillb", got)
}
