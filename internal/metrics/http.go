package metrics

import (
	"strconv"
	"time"
)

// HTTPRequestObserver records request middleware observations in Prometheus.
type HTTPRequestObserver struct{}

func (HTTPRequestObserver) ObserveRequest(method, path string, status int, duration time.Duration) {
	normalizedPath := NormalizePath(path)
	HTTPRequestsTotal.WithLabelValues(method, normalizedPath, strconv.Itoa(status)).Inc()
	HTTPRequestDuration.WithLabelValues(method, normalizedPath).Observe(duration.Seconds())
}
