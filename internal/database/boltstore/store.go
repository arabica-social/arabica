// Package boltstore provides persistent storage using BoltDB (bbolt).
// It implements the oauth.ClientAuthStore interface for session persistence
// and provides storage for the feed registry.
package boltstore

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Bucket names for organizing data
var (
	// BucketSessions stores OAuth session data keyed by "did:sessionID"
	BucketSessions = []byte("oauth_sessions")

	// BucketAuthRequests stores pending OAuth auth requests keyed by state
	BucketAuthRequests = []byte("oauth_auth_requests")

	// BucketFeedRegistry stores registered user DIDs for the community feed
	BucketFeedRegistry = []byte("feed_registry")

	// BucketModerationHiddenRecords stores AT-URIs of hidden records
	BucketModerationHiddenRecords = []byte("moderation_hidden_records")

	// BucketModerationBlacklist stores blacklisted user DIDs
	BucketModerationBlacklist = []byte("moderation_blacklist")

	// BucketModerationReports stores user reports on content
	BucketModerationReports = []byte("moderation_reports")

	// BucketModerationReportsByURI indexes reports by subject AT-URI
	BucketModerationReportsByURI = []byte("moderation_reports_by_uri")

	// BucketModerationReportsByDID indexes reports by subject DID
	BucketModerationReportsByDID = []byte("moderation_reports_by_did")

	// BucketModerationAuditLog stores moderation action audit trail
	BucketModerationAuditLog = []byte("moderation_audit_log")
)

// Store wraps a BoltDB database and provides access to specialized stores.
type Store struct {
	db *bolt.DB
}

// Options configures the BoltDB store.
type Options struct {
	// Path to the database file. Parent directories will be created if needed.
	Path string

	// Timeout for obtaining a file lock on the database.
	// If zero, a default of 5 seconds is used.
	Timeout time.Duration

	// FileMode for creating the database file.
	// If zero, 0600 is used.
	FileMode os.FileMode
}

// DefaultOptions returns sensible defaults for development.
func DefaultOptions() Options {
	return Options{
		Path:     "arabica.db",
		Timeout:  5 * time.Second,
		FileMode: 0600,
	}
}

// Open creates or opens a BoltDB database at the specified path.
// It creates all necessary buckets if they don't exist.
func Open(opts Options) (*Store, error) {
	if opts.Path == "" {
		opts.Path = "arabica.db"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}
	if opts.FileMode == 0 {
		opts.FileMode = 0600
	}

	// Ensure parent directory exists
	dir := filepath.Dir(opts.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open the database
	db, err := bolt.Open(opts.Path, opts.FileMode, &bolt.Options{
		Timeout: opts.Timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create buckets if they don't exist
	err = db.Update(func(tx *bolt.Tx) error {
		buckets := [][]byte{
			BucketSessions,
			BucketAuthRequests,
			BucketFeedRegistry,
			BucketModerationHiddenRecords,
			BucketModerationBlacklist,
			BucketModerationReports,
			BucketModerationReportsByURI,
			BucketModerationReportsByDID,
			BucketModerationAuditLog,
		}

		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists(bucket)
			if err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", bucket, err)
			}
		}

		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close closes the database.
func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// DB returns the underlying BoltDB instance for advanced operations.
func (s *Store) DB() *bolt.DB {
	return s.db
}

// SessionStore returns an OAuth session store backed by this database.
func (s *Store) SessionStore() *SessionStore {
	return &SessionStore{db: s.db}
}

// FeedStore returns a feed registry store backed by this database.
func (s *Store) FeedStore() *FeedStore {
	return &FeedStore{db: s.db}
}

// ModerationStore returns a moderation store backed by this database.
func (s *Store) ModerationStore() *ModerationStore {
	return &ModerationStore{db: s.db}
}

// Stats returns database statistics.
func (s *Store) Stats() bolt.Stats {
	return s.db.Stats()
}
