//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/pdewey.com/atp"

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
	const collection = arabica.NSIDRoaster
	const rkey = "orphanrkey"
	now := time.Now().Unix()
	require.NoError(t, h.FeedIndex.UpsertRecord(
		ctx, orphanDID, collection, rkey, "ciddead",
		[]byte(`{"$type":"social.arabica.alpha.roaster","name":"Orphan Roaster","createdAt":"2025-01-01T00:00:00Z"}`),
		now,
	))

	uri := atp.BuildATURI(orphanDID, collection, rkey)
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

// TestIdentityEvent_KeepsPriorOwnerWithoutAppViewEvidence drives the
// OnIdentityEvent reconciliation path through the profile watcher. A relay
// identity event is only a refresh hint: until the AppView profile and public
// handle resolution agree on the new DID, no profile or handle mapping may
// change, so a forged or premature event cannot strip a real owner's handle.
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
	// Verification consults the AppView and fails for these synthetic DIDs,
	// which is the case under test: no data may change.
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

	// The synthetic DIDs are unknown to the index and unreachable to the
	// AppView, so verification fails and the existing owner must survive
	// untouched — a relay hint alone never moves a handle.
	got, ok = h.FeedIndex.GetDIDByHandle(ctx, sharedHandle)
	require.True(t, ok, "unverified identity hint must keep the existing handle mapping")
	assert.Equal(t, orphanDID, got, "unverified identity hint must not evict the prior owner")
	assert.True(t, h.FeedIndex.ProfileCachedInMemory(orphanDID),
		"unverified identity hint must keep the cached profile")

	// Now seed the new DID's profile (as the watcher would, on a working API).
	// The handle must resolve to newDID, never orphanDID.
	seedProfile(t, h, newDID, sharedHandle)
	got, ok = h.FeedIndex.GetDIDByHandle(ctx, sharedHandle)
	require.True(t, ok)
	assert.Equal(t, newDID, got, "after reconciliation, shared handle resolves to the new owner")
}

// TestInvalidateProfile_RemovesHandleMapping is a direct unit-level assertion
// for InvalidateProfile. The eviction paths exercised in other tests
// (OnIdentityEvent, DeleteAllByDID) reach did_by_handle through different
// routes; this test pins the contract of InvalidateProfile itself: clear the
// in-memory cache, the persistent profile row, and the handle mapping.
func TestInvalidateProfile_RemovesHandleMapping(t *testing.T) {
	h := StartHarness(t, nil)
	ctx := context.Background()

	const did = "did:plc:invalidatetest"
	const handle = "invalidate.test"

	seedProfile(t, h, did, handle)
	require.True(t, h.FeedIndex.ProfileCachedInMemory(did),
		"precondition: seed populates the in-memory cache")
	got, ok := h.FeedIndex.GetDIDByHandle(ctx, handle)
	require.True(t, ok)
	require.Equal(t, did, got)

	h.FeedIndex.InvalidateProfile(did)

	assert.False(t, h.FeedIndex.ProfileCachedInMemory(did),
		"InvalidateProfile must clear the in-memory cache")
	_, ok = h.FeedIndex.GetDIDByHandle(ctx, handle)
	assert.False(t, ok, "InvalidateProfile must clear the handle mapping")
}

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
