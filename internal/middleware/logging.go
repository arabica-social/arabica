package middleware

import (
	"net/http"
	"time"

	"arabica/internal/atproto"

	"github.com/rs/zerolog"
)

// LoggingMiddleware returns a middleware that logs HTTP request details with structured logging
func LoggingMiddleware(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer wrapper to capture status code and bytes written
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
				bytesWritten:   0,
			}

			// Call the next handler
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Build structured log event
			logEvent := logger.Info().
				Str("method", r.Method).
				Str("path", r.URL.Path).
				Str("query", r.URL.RawQuery).
				Int("status", rw.statusCode).
				Dur("duration", duration).
				Str("remote_addr", r.RemoteAddr).
				Str("user_agent", r.UserAgent()).
				Int64("bytes_written", rw.bytesWritten).
				Str("proto", r.Proto)

			// Add referer if present
			if referer := r.Referer(); referer != "" {
				logEvent.Str("referer", referer)
			}

			// Add request ID if present (could be added by another middleware)
			if reqID := r.Header.Get("X-Request-ID"); reqID != "" {
				logEvent.Str("request_id", reqID)
			}

			// Add content type if present
			if contentType := r.Header.Get("Content-Type"); contentType != "" {
				logEvent.Str("content_type", contentType)
			}

			// Add authenticated user DID if present
			if did, err := atproto.GetAuthenticatedDID(r.Context()); err == nil && did != "" {
				logEvent.Str("user_did", did)
			}

			// Change log level based on status code
			if rw.statusCode >= 500 {
				logEvent = logger.Error().Fields(logEvent)
			} else if rw.statusCode >= 400 {
				logEvent = logger.Warn().Fields(logEvent)
			}

			logEvent.Msg("HTTP request")
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture the status code and bytes written
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}
