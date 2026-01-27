package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// TestKnownDIDsLogging verifies that DIDs are logged correctly
func TestKnownDIDsLogging(t *testing.T) {
	// Create a buffer to capture log output
	var buf bytes.Buffer

	// Configure zerolog to write JSON to our buffer
	originalLogger := log.Logger
	defer func() {
		log.Logger = originalLogger
	}()

	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-dids.txt")

	content := `# Test DIDs
did:plc:abc123
did:web:example.com
did:plc:xyz789
`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load DIDs from file
	dids, err := loadKnownDIDs(testFile)
	if err != nil {
		t.Fatalf("loadKnownDIDs failed: %v", err)
	}

	// Simulate logging (like we do in main.go)
	log.Info().
		Int("count", len(dids)).
		Str("file", testFile).
		Strs("dids", dids).
		Msg("Loaded known DIDs from file")

	// Parse the log output
	logOutput := buf.String()

	// Verify it contains JSON log
	if !strings.Contains(logOutput, "Loaded known DIDs from file") {
		t.Errorf("Log output missing expected message. Got: %s", logOutput)
	}

	// Parse as JSON to verify structure
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry); err != nil {
		t.Fatalf("Failed to parse log as JSON: %v\nOutput: %s", err, logOutput)
	}

	// Verify log fields
	if logEntry["count"] != float64(3) {
		t.Errorf("Expected count=3, got %v", logEntry["count"])
	}

	if logEntry["file"] != testFile {
		t.Errorf("Expected file=%s, got %v", testFile, logEntry["file"])
	}

	// Verify DIDs array is present
	didsFromLog, ok := logEntry["dids"].([]interface{})
	if !ok {
		t.Fatalf("Expected 'dids' to be an array, got %T", logEntry["dids"])
	}

	if len(didsFromLog) != 3 {
		t.Errorf("Expected 3 DIDs in log, got %d", len(didsFromLog))
	}

	// Verify DID values
	expectedDIDs := map[string]bool{
		"did:plc:abc123":      false,
		"did:web:example.com": false,
		"did:plc:xyz789":      false,
	}

	for _, did := range didsFromLog {
		didStr, ok := did.(string)
		if !ok {
			t.Errorf("DID is not a string: %v", did)
			continue
		}
		if _, exists := expectedDIDs[didStr]; exists {
			expectedDIDs[didStr] = true
		} else {
			t.Errorf("Unexpected DID in log: %s", didStr)
		}
	}

	for did, found := range expectedDIDs {
		if !found {
			t.Errorf("Expected DID not found in log: %s", did)
		}
	}
}
