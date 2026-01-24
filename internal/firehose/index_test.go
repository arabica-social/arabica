package firehose

import (
	"testing"
	"time"
)

func TestBackfillTracking(t *testing.T) {
	// Create temporary index
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer idx.Close()

	testDID := "did:plc:test123abc"

	// Initially should not be backfilled
	if idx.IsBackfilled(testDID) {
		t.Error("DID should not be backfilled initially")
	}

	// Mark as backfilled
	if err := idx.MarkBackfilled(testDID); err != nil {
		t.Fatalf("Failed to mark DID as backfilled: %v", err)
	}

	// Now should be backfilled
	if !idx.IsBackfilled(testDID) {
		t.Error("DID should be marked as backfilled")
	}

	// Different DID should not be backfilled
	otherDID := "did:plc:other456def"
	if idx.IsBackfilled(otherDID) {
		t.Error("Other DID should not be backfilled")
	}
}

func TestBackfillTracking_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"
	testDID := "did:plc:persist123"

	// Create index and mark DID as backfilled
	{
		idx, err := NewFeedIndex(dbPath, 1*time.Hour)
		if err != nil {
			t.Fatalf("Failed to create index: %v", err)
		}

		if err := idx.MarkBackfilled(testDID); err != nil {
			t.Fatalf("Failed to mark DID as backfilled: %v", err)
		}

		idx.Close()
	}

	// Reopen index and verify DID is still marked as backfilled
	{
		idx, err := NewFeedIndex(dbPath, 1*time.Hour)
		if err != nil {
			t.Fatalf("Failed to reopen index: %v", err)
		}
		defer idx.Close()

		if !idx.IsBackfilled(testDID) {
			t.Error("DID should still be marked as backfilled after reopening")
		}
	}
}

func TestBackfillTracking_MultipleDIDs(t *testing.T) {
	tmpDir := t.TempDir()
	idx, err := NewFeedIndex(tmpDir+"/test.db", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create index: %v", err)
	}
	defer idx.Close()

	dids := []string{
		"did:plc:user1",
		"did:plc:user2",
		"did:web:example.com",
		"did:plc:user3",
	}

	// Mark all as backfilled
	for _, did := range dids {
		if err := idx.MarkBackfilled(did); err != nil {
			t.Fatalf("Failed to mark DID %s as backfilled: %v", did, err)
		}
	}

	// Verify all are marked
	for _, did := range dids {
		if !idx.IsBackfilled(did) {
			t.Errorf("DID %s should be marked as backfilled", did)
		}
	}
}
