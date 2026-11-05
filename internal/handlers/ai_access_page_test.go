package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAIDeniedPageResponse(t *testing.T) {
	h := AIHandlers{Handler: &Handler{}}
	for _, status := range []int{0, http.StatusOK, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			shell := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-cache")
				if status != 0 {
					w.WriteHeader(status)
				}
				_, _ = w.Write([]byte("<html>app shell</html>"))
			})
			rec := httptest.NewRecorder()
			h.RequirePageAccess(shell).ServeHTTP(rec, httptest.NewRequest("GET", "/chat", nil))
			expected := http.StatusNotFound
			if status == http.StatusInternalServerError {
				expected = status
			}
			result := rec.Result()
			require.Equal(t, expected, result.StatusCode)
			require.Equal(t, "text/html; charset=utf-8", result.Header.Get("Content-Type"))
			require.Equal(t, "no-store", result.Header.Get("Cache-Control"))
			require.Equal(t, "<html>app shell</html>", rec.Body.String())
		})
	}
}
