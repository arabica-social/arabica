package metrics

import (
	"strings"

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

// Witness cache metrics
var (
	WitnessCacheHitsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_witness_cache_hits_total",
		Help: "Total witness cache hits (PDS request avoided)",
	}, []string{"collection"})

	WitnessCacheMissesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_witness_cache_misses_total",
		Help: "Total witness cache misses (fell back to PDS)",
	}, []string{"collection"})
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

// SQLite metrics
var (
	SQLiteOpenConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_sqlite_open_connections",
		Help: "Number of open SQLite connections in the pool",
	})

	SQLiteInUse = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_sqlite_connections_in_use",
		Help: "Number of SQLite connections currently in use",
	})

	SQLiteWaitCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_sqlite_wait_count_total",
		Help: "Total number of times a goroutine had to wait for a SQLite connection (cumulative since process start)",
	})

	SQLiteWaitDurationSeconds = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "arabica_sqlite_wait_duration_seconds_total",
		Help: "Total time spent waiting for SQLite connections in seconds (cumulative since process start)",
	})
)

// RouteLabel returns a bounded path label from the pattern selected by
// http.ServeMux. Call it after the handler has returned, when Request.Pattern
// has been populated. Unknown requests share one label rather than using the
// raw URL path.
func RouteLabel(pattern string) string {
	if _, route, ok := strings.Cut(pattern, " "); ok {
		pattern = route
	}

	switch {
	case pattern == "", pattern == "/":
		return "/{unmatched}"
	case pattern == "/{$}":
		return "/"
	case strings.HasPrefix(pattern, "/static/"):
		return "/static/*"
	case strings.HasPrefix(pattern, "/_app/"):
		return "/_app/*"
	default:
		return pattern
	}
}

// HTTPMethod returns a bounded request method label. HTTP methods are arbitrary
// tokens on the wire, so recording an unknown token verbatim would let callers
// create series just as raw paths did.
func HTTPMethod(method string) string {
	switch method {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS":
		return method
	default:
		return "OTHER"
	}
}

// HTTPSpanName returns the low-cardinality span name used for HTTP server
// spans. Both the method and route portions are bounded.
func HTTPSpanName(method, pattern string) string {
	return HTTPMethod(method) + " " + RouteLabel(pattern)
}
