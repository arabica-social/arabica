package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP metrics
var (
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "arabica_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
	}, []string{"method", "path"})
)

// Firehose metrics
var (
	FirehoseEventsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_firehose_events_total",
		Help: "Total number of firehose events processed",
	}, []string{"collection", "operation"})

	FirehoseConnectionState = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_firehose_connection_state",
		Help: "Firehose connection state (1=connected, 0=disconnected)",
	})

	FirehoseErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "arabica_firehose_errors_total",
		Help: "Total number of firehose processing errors",
	})
)

// PDS metrics
var (
	PDSRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_pds_requests_total",
		Help: "Total number of PDS requests",
	}, []string{"method", "collection"})
)

// Feed metrics
var (
	FeedCacheHitsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "arabica_feed_cache_hits_total",
		Help: "Total number of feed cache hits",
	})

	FeedCacheMissesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "arabica_feed_cache_misses_total",
		Help: "Total number of feed cache misses",
	})
)

// Business metrics (gauges updated periodically by collector)
var (
	KnownUsersTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_known_users_total",
		Help: "Total number of unique DIDs in the index",
	})

	RegisteredUsersTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_registered_users_total",
		Help: "Total number of registered feed users",
	})

	IndexedRecordsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_indexed_records_total",
		Help: "Total number of indexed records",
	})

	JoinRequestsPending = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_join_requests_pending",
		Help: "Number of pending join requests",
	})

	IndexedLikesTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_indexed_likes_total",
		Help: "Total number of indexed likes",
	})

	IndexedCommentsTotal = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_indexed_comments_total",
		Help: "Total number of indexed comments",
	})

	IndexedRecordsByCollection = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "arabica_indexed_records_by_collection",
		Help: "Number of indexed records by collection type",
	}, []string{"collection"})
)

// Event counters (incremented on occurrence)
var (
	AuthLoginsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_auth_logins_total",
		Help: "Total number of login attempts",
	}, []string{"status"})

	JoinRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "arabica_join_requests_total",
		Help: "Total number of join request submissions",
	})

	InvitesCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "arabica_invites_created_total",
		Help: "Total number of invites created",
	})

	LikesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_likes_total",
		Help: "Total number of like operations",
	}, []string{"operation"})

	CommentsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_comments_total",
		Help: "Total number of comment operations",
	}, []string{"operation"})

	ReportsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "arabica_reports_total",
		Help: "Total number of user reports submitted",
	})
)

// NormalizePath reduces high-cardinality path labels by replacing dynamic
// segments with placeholders. This keeps the metric label space bounded.
func NormalizePath(path string) string {
	// Static assets - collapse into one label
	if len(path) > 8 && path[:8] == "/static/" {
		return "/static/*"
	}

	// Known patterns with IDs - normalize the dynamic segments
	// Routes like /brews/{id}, /beans/{id}, /profile/{actor}, etc.
	segments := splitPath(path)
	if len(segments) < 2 {
		return path
	}

	switch segments[0] {
	case "brews":
		if len(segments) == 2 {
			if segments[1] == "new" || segments[1] == "export" {
				return path
			}
			return "/brews/:id"
		}
		if len(segments) == 3 && segments[2] == "edit" {
			return "/brews/:id/edit"
		}
	case "beans", "roasters", "grinders", "brewers":
		if len(segments) == 2 {
			return "/" + segments[0] + "/:id"
		}
	case "profile":
		if len(segments) == 2 {
			return "/profile/:actor"
		}
	case "api":
		if len(segments) >= 3 {
			switch segments[1] {
			case "beans", "roasters", "grinders", "brewers", "comments":
				if len(segments) == 3 {
					return "/api/" + segments[1] + "/:id"
				}
			case "profile":
				if len(segments) == 3 {
					return "/api/profile/:actor"
				}
			case "modals":
				if len(segments) == 4 {
					if segments[3] == "new" {
						return "/api/modals/" + segments[2] + "/new"
					}
					return "/api/modals/" + segments[2] + "/:id"
				}
			}
		}
	}

	return path
}

func splitPath(path string) []string {
	// Skip leading slash
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	// Split on /
	var segments []string
	start := 0
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				segments = append(segments, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		segments = append(segments, path[start:])
	}
	return segments
}
