package metrics

import (
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	provider "tangled.org/arabica.social/arabica/internal/ai"
)

var (
	// AIRequestLogErrorsTotal counts calls whose per-account history could not be persisted.
	AIRequestLogErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "arabica_ai_request_log_errors_total",
		Help: "Failed AI request log writes; per-account usage history is incomplete.",
	})
	aiRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_ai_requests_total",
		Help: "Attempted upstream AI calls by configured provider/model, outcome and HTTP status (0 when no response).",
	}, []string{"provider", "model", "outcome", "status"})
	aiDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "arabica_ai_request_duration_seconds",
		Help:    "Upstream AI call duration including response reading and decoding.",
		Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10, 20, 30, 60, 120},
	}, []string{"provider", "model"})
	aiTokens = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_ai_tokens_total",
		Help: "Provider-reported AI tokens; input includes cached_input, output includes reasoning tokens.",
	}, []string{"provider", "model", "type"})
	aiUsage = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "arabica_ai_usage_reports_total",
		Help: "AI usage reporting coverage: reported, missing, invalid, or unavailable (no decodable 2xx body).",
	}, []string{"provider", "model", "state"})
)

// ObserveAI exports transport observations without user or conversation labels.
func ObserveAI(o provider.Observation) {
	outcome := o.Failure.Category
	if outcome == "" {
		outcome = "success"
	}
	aiRequests.WithLabelValues(o.Provider, o.Model, outcome, strconv.Itoa(o.HTTPStatus)).Inc()
	aiDuration.WithLabelValues(o.Provider, o.Model).Observe(o.Duration.Seconds())
	aiUsage.WithLabelValues(o.Provider, o.Model, o.UsageStatus).Inc()
	if o.Usage != nil {
		aiTokens.WithLabelValues(o.Provider, o.Model, "input").Add(float64(o.Usage.InputTokens))
		aiTokens.WithLabelValues(o.Provider, o.Model, "output").Add(float64(o.Usage.OutputTokens))
		aiTokens.WithLabelValues(o.Provider, o.Model, "cached_input").Add(float64(o.Usage.CachedInputTokens))
	}
}
