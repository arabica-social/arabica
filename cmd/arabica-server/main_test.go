package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadKnownDIDs(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-dids.txt")

	content := `# This is a comment
did:plc:test123abc
did:web:example.com

# Another comment

did:plc:another456def

# Invalid lines below
not-a-did
just some text

# Valid DID after invalid ones
did:plc:final789ghi
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test loading DIDs
	dids, err := loadKnownDIDs(testFile)
	if err != nil {
		t.Fatalf("loadKnownDIDs failed: %v", err)
	}

	// Expected DIDs
	expected := []string{
		"did:plc:test123abc",
		"did:web:example.com",
		"did:plc:another456def",
		"did:plc:final789ghi",
	}

	if len(dids) != len(expected) {
		t.Errorf("Expected %d DIDs, got %d", len(expected), len(dids))
	}

	for i, expectedDID := range expected {
		if i >= len(dids) {
			t.Errorf("Missing DID at index %d: %s", i, expectedDID)
			continue
		}
		if dids[i] != expectedDID {
			t.Errorf("DID at index %d: expected %s, got %s", i, expectedDID, dids[i])
		}
	}
}

func TestLoadKnownDIDs_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	dids, err := loadKnownDIDs(testFile)
	if err != nil {
		t.Fatalf("loadKnownDIDs failed: %v", err)
	}

	if len(dids) != 0 {
		t.Errorf("Expected 0 DIDs from empty file, got %d", len(dids))
	}
}

func TestLoadKnownDIDs_OnlyComments(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "comments.txt")

	content := `# Comment 1
# Comment 2

# Comment 3
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	dids, err := loadKnownDIDs(testFile)
	if err != nil {
		t.Fatalf("loadKnownDIDs failed: %v", err)
	}

	if len(dids) != 0 {
		t.Errorf("Expected 0 DIDs from comments-only file, got %d", len(dids))
	}
}

func TestLoadKnownDIDs_NonexistentFile(t *testing.T) {
	_, err := loadKnownDIDs("/nonexistent/path/file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}
