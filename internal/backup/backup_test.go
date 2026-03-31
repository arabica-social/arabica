package backup

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSQLiteBackup(t *testing.T) {
	// Create a temp SQLite DB with some data.
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY, val TEXT)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO t (val) VALUES ('hello'), ('world')")
	require.NoError(t, err)

	// Backup via VACUUM INTO.
	src := NewSQLiteSource("test-db", db)
	assert.Equal(t, "test-db", src.Name())

	backupPath := filepath.Join(t.TempDir(), "backup.db")
	err = src.Backup(context.Background(), backupPath)
	require.NoError(t, err)

	// Verify the backup is a valid SQLite DB with the data.
	backupDB, err := sql.Open("sqlite", backupPath)
	require.NoError(t, err)
	defer backupDB.Close()

	var count int
	err = backupDB.QueryRow("SELECT count(*) FROM t").Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)
}

func TestLocalDestinationRetention(t *testing.T) {
	dir := t.TempDir()
	dest, err := NewLocalDestination(dir)
	require.NoError(t, err)

	ctx := context.Background()

	// Create 4 fake backup files.
	for _, name := range []string{
		"db-20260101-000000.bak",
		"db-20260102-000000.bak",
		"db-20260103-000000.bak",
		"db-20260104-000000.bak",
	} {
		tmpFile := filepath.Join(t.TempDir(), name)
		require.NoError(t, os.WriteFile(tmpFile, []byte("data"), 0644))
		require.NoError(t, dest.Write(ctx, name, tmpFile))
	}

	// List should return newest first.
	keys, err := dest.List(ctx, "db-")
	require.NoError(t, err)
	assert.Equal(t, []string{
		"db-20260104-000000.bak",
		"db-20260103-000000.bak",
		"db-20260102-000000.bak",
		"db-20260101-000000.bak",
	}, keys)

	// Delete oldest two.
	require.NoError(t, dest.Delete(ctx, "db-20260101-000000.bak"))
	require.NoError(t, dest.Delete(ctx, "db-20260102-000000.bak"))

	keys, err = dest.List(ctx, "db-")
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestServicePrunesOldBackups(t *testing.T) {
	// Set up a real SQLite DB.
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("CREATE TABLE t (id INTEGER PRIMARY KEY)")
	require.NoError(t, err)

	backupDir := t.TempDir()
	dest, err := NewLocalDestination(backupDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Pre-seed 3 old backups.
	for _, name := range []string{
		"test-20260101-000000.bak",
		"test-20260102-000000.bak",
		"test-20260103-000000.bak",
	} {
		require.NoError(t, os.WriteFile(filepath.Join(backupDir, name), []byte("old"), 0644))
	}

	svc := NewService(Config{
		Retain: 2,
		Dest:   dest,
	})
	svc.AddSource(NewSQLiteSource("test", db))

	// One new backup run — should create 1 new + prune old to keep only 2 total.
	svc.runAll(ctx)

	keys, err := dest.List(ctx, "test-")
	require.NoError(t, err)
	assert.Equal(t, 2, len(keys))
}

func TestNextOccurrence(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		hour     int
		expected time.Time
	}{
		{
			name:     "hour hasn't passed yet today",
			now:      time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC),
			hour:     11,
			expected: time.Date(2026, 3, 31, 11, 0, 0, 0, time.UTC),
		},
		{
			name:     "hour already passed today",
			now:      time.Date(2026, 3, 31, 14, 0, 0, 0, time.UTC),
			hour:     11,
			expected: time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC),
		},
		{
			name:     "exactly at the scheduled hour",
			now:      time.Date(2026, 3, 31, 11, 0, 0, 0, time.UTC),
			hour:     11,
			expected: time.Date(2026, 4, 1, 11, 0, 0, 0, time.UTC),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nextOccurrence(tt.now, tt.hour)
			assert.Equal(t, tt.expected, result)
		})
	}
}
