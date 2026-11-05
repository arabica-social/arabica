package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBodyLimitRouteOverrides(t *testing.T) {
	const largerLimit = 2 << 20
	consume := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.Copy(io.Discard, r.Body)
		if err != nil {
			var oversized *http.MaxBytesError
			require.True(t, errors.As(err, &oversized))
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})
	for _, tc := range []struct {
		name, pattern, contentType string
		size, status               int
	}{
		{"default boundary", "POST /ordinary", "application/json", MaxJSONBodySize, 204},
		{"default overflow", "POST /ordinary", "application/json", MaxJSONBodySize + 1, 413},
		{"override boundary", "POST /larger", "application/json", largerLimit, 204},
		{"override overflow", "POST /larger", "application/json", largerLimit + 1, 413},
		{"different method", "PUT /larger", "application/json", MaxJSONBodySize + 1, 413},
		{"unmatched path", "", "application/json", MaxJSONBodySize + 1, 413},
		{"form unchanged", "POST /ordinary", "application/x-www-form-urlencoded", MaxFormBodySize + 1, 413},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/larger", strings.NewReader(strings.Repeat("x", tc.size)))
			req.Pattern = tc.pattern
			req.Header.Set("Content-Type", tc.contentType)
			rec := httptest.NewRecorder()
			LimitBodyMiddlewareWithRouteLimits(consume, map[string]int64{"POST /larger": largerLimit}).ServeHTTP(rec, req)
			require.Equal(t, tc.status, rec.Code)
		})
	}
}
