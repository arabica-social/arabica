package ai

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	"tangled.org/arabica.social/arabica/internal/metrics"
)

type RequestLogStore struct{ db *sql.DB }

func NewRequestLogStore(db *sql.DB) (*RequestLogStore, error) {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS ai_request_log (
  id INTEGER PRIMARY KEY,
  started_at_ms INTEGER NOT NULL,
  did TEXT NOT NULL,
  request_id TEXT NOT NULL,
  trace_id TEXT NOT NULL,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  duration_ns INTEGER NOT NULL CHECK (duration_ns >= 0),
  outcome TEXT NOT NULL,
  http_status INTEGER NOT NULL,
  usage_state TEXT NOT NULL,
  input_tokens INTEGER,
  output_tokens INTEGER,
  cached_input_tokens INTEGER,
  CHECK ((usage_state = 'reported' AND input_tokens >= 0 AND output_tokens >= 0
    AND cached_input_tokens >= 0 AND cached_input_tokens <= input_tokens
    AND input_tokens IS NOT NULL AND output_tokens IS NOT NULL AND cached_input_tokens IS NOT NULL)
   OR (usage_state IN ('missing', 'invalid', 'unavailable') AND input_tokens IS NULL
    AND output_tokens IS NULL AND cached_input_tokens IS NULL))
 )`); err != nil {
		return nil, err
	}
	for _, statement := range []string{
		`CREATE INDEX IF NOT EXISTS ai_request_log_did_started ON ai_request_log (did, started_at_ms)`,
		`CREATE INDEX IF NOT EXISTS ai_request_log_started ON ai_request_log (started_at_ms)`,
		`CREATE INDEX IF NOT EXISTS ai_request_log_request ON ai_request_log (did, request_id)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			return nil, err
		}
	}
	return &RequestLogStore{db: db}, nil
}

func (s *RequestLogStore) Record(ctx context.Context, did, requestID, traceID string, o provider.Observation) error {
	if did == "" || requestID == "" || o.StartedAt.IsZero() {
		return errors.New("ai request log: missing identity or timestamp")
	}
	outcome := o.Failure.Category
	if outcome == "" {
		outcome = "success"
	}
	var input, output, cached any
	if o.Usage != nil {
		input, output, cached = o.Usage.InputTokens, o.Usage.OutputTokens, o.Usage.CachedInputTokens
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO ai_request_log
  (started_at_ms, did, request_id, trace_id, provider, model, duration_ns, outcome,
   http_status, usage_state, input_tokens, output_tokens, cached_input_tokens)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		o.StartedAt.UnixMilli(), did, requestID, traceID, o.Provider, o.Model, o.Duration.Nanoseconds(),
		outcome, o.HTTPStatus, o.UsageStatus, input, output, cached)
	return err
}

// SetRequestLog configures persistence once during server bootstrap. A nil store
// leaves request logging disabled, independently of quotas and Prometheus.
func (s *Service) SetRequestLog(store *RequestLogStore) { s.requestLog = store }

func (s *Service) withRequestLog(ctx context.Context, did, requestID string) context.Context {
	if s.requestLog == nil {
		return ctx
	}
	traceID := ""
	if span := trace.SpanContextFromContext(ctx); span.IsValid() {
		traceID = span.TraceID().String()
	}
	return provider.WithRequestObserver(ctx, func(o provider.Observation) {
		// Client cancellation must not erase the usage of a potentially billed call.
		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		if err := s.requestLog.Record(writeCtx, did, requestID, traceID, o); err != nil {
			metrics.AIRequestLogErrorsTotal.Inc()
			log.Error().Msg("AI request log write failed; usage history is incomplete")
		}
	})
}
