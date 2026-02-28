// Package firehose provides real-time AT Protocol event consumption via Jetstream.
// It indexes Arabica records into a local SQLite database for fast feed queries.
package firehose

import (
	"arabica/internal/atproto"
)

// Default Jetstream public endpoints
var DefaultJetstreamEndpoints = []string{
	"wss://jetstream1.us-east.bsky.network/subscribe",
	"wss://jetstream2.us-east.bsky.network/subscribe",
	"wss://jetstream1.us-west.bsky.network/subscribe",
	"wss://jetstream2.us-west.bsky.network/subscribe",
}

// ArabicaCollections lists all Arabica lexicon collections to filter for
var ArabicaCollections = []string{
	atproto.NSIDBrew,
	atproto.NSIDBean,
	atproto.NSIDRoaster,
	atproto.NSIDGrinder,
	atproto.NSIDBrewer,
	atproto.NSIDLike,
	atproto.NSIDComment,
}

// Config holds configuration for the Jetstream consumer
type Config struct {
	// Endpoints is a list of Jetstream WebSocket URLs to connect to (with fallback rotation)
	Endpoints []string

	// WantedCollections filters events to specific collection NSIDs
	WantedCollections []string

	// Compress enables zstd compression (~56% bandwidth reduction)
	Compress bool

	// IndexPath is the path to the SQLite feed index database
	IndexPath string

	// ProfileCacheTTL is how long to cache profile data
	ProfileCacheTTL int64 // seconds
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Endpoints:         DefaultJetstreamEndpoints,
		WantedCollections: ArabicaCollections,
		Compress:          false, // Disabled: Jetstream uses custom zstd dictionary
		IndexPath:         "",    // Will be set based on data directory
		ProfileCacheTTL:   3600,  // 1 hour
	}
}
