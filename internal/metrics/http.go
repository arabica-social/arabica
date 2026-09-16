package metrics

import (
	"strconv"
	"time"
)

// HTTPRequestObserver records request middleware observations in Prometheus.
type HTTPRequestObserver struct{}

func (HTTPRequestObserver) ObserveRequest(method, routePattern string, status int, duration time.Duration) {
	route := RouteLabel(routePattern)
	method = HTTPMethod(method)
	HTTPRequestsTotal.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	HTTPRequestDuration.WithLabelValues(method, route).Observe(duration.Seconds())
}
