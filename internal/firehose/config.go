// Package firehose provides real-time AT Protocol event consumption via Jetstream.
// It indexes records into a local SQLite database for fast feed queries.
package firehose

// Default Jetstream public endpoints
var DefaultJetstreamEndpoints = []string{
	"wss://jetstream1.us-east.bsky.network/subscribe",
	"wss://jetstream2.us-east.bsky.network/subscribe",
	"wss://jetstream1.us-west.bsky.network/subscribe",
	"wss://jetstream2.us-west.bsky.network/subscribe",
}

// NSIDBlueskyProfile is the AT Protocol collection for user profile records.
// Watched by ProfileWatcher (separate connection) for known users only.
const NSIDBlueskyProfile = "app.bsky.actor.profile"

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

// DefaultConfig returns a configuration with sensible defaults. Caller
// must populate WantedCollections (typically from domain.App.NSIDs()) — a
// nil default forces app-aware wiring at startup so the subscription tracks
// the running app's entity set.
func DefaultConfig() *Config {
	return &Config{
		Endpoints:         DefaultJetstreamEndpoints,
		WantedCollections: nil,
		Compress:          false, // Disabled: Jetstream uses custom zstd dictionary
		IndexPath:         "",    // Will be set based on data directory
		ProfileCacheTTL:   3600,  // 1 hour
	}
}
