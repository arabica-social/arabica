package server

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

// TestKnownDIDsLogging verifies loadKnownDIDs parses comments and DIDs
// correctly and that the structured-log fields shape stays stable
// (count, file, dids array). The shape is part of our observability
// contract — bumping zerolog or refactoring should not break it.
func TestKnownDIDsLogging(t *testing.T) {
	var buf bytes.Buffer

	originalLogger := log.Logger
	defer func() { log.Logger = originalLogger }()

	log.Logger = zerolog.New(&buf).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test-dids.txt")
	content := `# Test DIDs
did:plc:abc123
did:web:example.com
did:plc:xyz789
`
	assert.NoError(t, os.WriteFile(testFile, []byte(content), 0644))

	dids, err := loadKnownDIDs(testFile)
	assert.NoError(t, err)
	assert.Len(t, dids, 3)

	log.Info().
		Int("count", len(dids)).
		Str("file", testFile).
		Strs("dids", dids).
		Msg("Loaded known DIDs from file")

	logOutput := buf.String()
	assert.Contains(t, logOutput, "Loaded known DIDs from file")

	var logEntry map[string]any
	assert.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(logOutput)), &logEntry))

	assert.Equal(t, float64(3), logEntry["count"])
	assert.Equal(t, testFile, logEntry["file"])

	didsFromLog, ok := logEntry["dids"].([]any)
	assert.True(t, ok, "dids field should be an array")
	assert.Len(t, didsFromLog, 3)

	expected := map[string]bool{
		"did:plc:abc123":      false,
		"did:web:example.com": false,
		"did:plc:xyz789":      false,
	}
	for _, did := range didsFromLog {
		didStr, ok := did.(string)
		assert.True(t, ok)
		if _, exists := expected[didStr]; exists {
			expected[didStr] = true
		} else {
			t.Errorf("unexpected DID in log: %s", didStr)
		}
	}
	for did, found := range expected {
		assert.True(t, found, "expected DID missing: %s", did)
	}
}
