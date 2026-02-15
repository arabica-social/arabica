package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify nonce is available in context
		nonce := CSPNonceFromContext(r.Context())
		assert.NotEmpty(t, nonce, "nonce should be set in context")
		w.WriteHeader(http.StatusOK)
	})

	wrapped := SecurityHeadersMiddleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), microphone=(), camera=()", rec.Header().Get("Permissions-Policy"))

	csp := rec.Header().Get("Content-Security-Policy")
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src 'self' 'unsafe-eval' 'nonce-")
	assert.Contains(t, csp, "frame-ancestors 'none'")
}

func TestCSPNonceFromContext(t *testing.T) {
	t.Run("returns nonce when set", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), cspNonceKey, "test-nonce-123")
		assert.Equal(t, "test-nonce-123", CSPNonceFromContext(ctx))
	})

	t.Run("returns empty string when not set", func(t *testing.T) {
		assert.Equal(t, "", CSPNonceFromContext(context.Background()))
	})

	t.Run("returns empty string for wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), cspNonceKey, 12345)
		assert.Equal(t, "", CSPNonceFromContext(ctx))
	})
}

func TestCSPNonceUniqueness(t *testing.T) {
	nonces := make(map[string]bool)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := CSPNonceFromContext(r.Context())
		nonces[nonce] = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := SecurityHeadersMiddleware(handler)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
	}

	assert.Len(t, nonces, 10, "each request should get a unique nonce")
}

func TestRateLimiter_Allow(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		rl := &RateLimiter{
			visitors: make(map[string]*visitor),
			rate:     3,
			window:   time.Minute,
			cleanup:  2 * time.Minute,
		}

		assert.True(t, rl.Allow("192.168.1.1"))
		assert.True(t, rl.Allow("192.168.1.1"))
		assert.True(t, rl.Allow("192.168.1.1"))
	})

	t.Run("blocks after exceeding limit", func(t *testing.T) {
		rl := &RateLimiter{
			visitors: make(map[string]*visitor),
			rate:     2,
			window:   time.Minute,
			cleanup:  2 * time.Minute,
		}

		assert.True(t, rl.Allow("10.0.0.1"))
		assert.True(t, rl.Allow("10.0.0.1"))
		assert.False(t, rl.Allow("10.0.0.1"))
	})

	t.Run("different IPs are independent", func(t *testing.T) {
		rl := &RateLimiter{
			visitors: make(map[string]*visitor),
			rate:     1,
			window:   time.Minute,
			cleanup:  2 * time.Minute,
		}

		assert.True(t, rl.Allow("10.0.0.1"))
		assert.False(t, rl.Allow("10.0.0.1"))
		assert.True(t, rl.Allow("10.0.0.2"))
	})

	t.Run("resets after window expires", func(t *testing.T) {
		rl := &RateLimiter{
			visitors: make(map[string]*visitor),
			rate:     1,
			window:   50 * time.Millisecond,
			cleanup:  100 * time.Millisecond,
		}

		assert.True(t, rl.Allow("10.0.0.1"))
		assert.False(t, rl.Allow("10.0.0.1"))

		time.Sleep(60 * time.Millisecond)
		assert.True(t, rl.Allow("10.0.0.1"))
	})
}

func TestRateLimitMiddleware(t *testing.T) {
	config := &RateLimitConfig{
		AuthLimiter:   &RateLimiter{visitors: make(map[string]*visitor), rate: 2, window: time.Minute, cleanup: 2 * time.Minute},
		APILimiter:    &RateLimiter{visitors: make(map[string]*visitor), rate: 3, window: time.Minute, cleanup: 2 * time.Minute},
		GlobalLimiter: &RateLimiter{visitors: make(map[string]*visitor), rate: 5, window: time.Minute, cleanup: 2 * time.Minute},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := RateLimitMiddleware(config)
	wrapped := middleware(handler)

	t.Run("auth endpoints use auth limiter", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
			req.RemoteAddr = "1.1.1.1:1234"
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}

		req := httptest.NewRequest(http.MethodPost, "/auth/login", nil)
		req.RemoteAddr = "1.1.1.1:1234"
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Equal(t, "60", rec.Header().Get("Retry-After"))
	})

	t.Run("api endpoints use api limiter", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			req := httptest.NewRequest(http.MethodGet, "/api/brews", nil)
			req.RemoteAddr = "2.2.2.2:1234"
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/brews", nil)
		req.RemoteAddr = "2.2.2.2:1234"
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})

	t.Run("other endpoints use global limiter", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/brews", nil)
			req.RemoteAddr = "3.3.3.3:1234"
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}

		req := httptest.NewRequest(http.MethodGet, "/brews", nil)
		req.RemoteAddr = "3.3.3.3:1234"
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})

	t.Run("login path uses auth limiter", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = "4.4.4.4:1234"
			rec := httptest.NewRecorder()
			wrapped.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code)
		}

		req := httptest.NewRequest(http.MethodPost, "/login", nil)
		req.RemoteAddr = "4.4.4.4:1234"
		rec := httptest.NewRecorder()
		wrapped.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})
}

func TestRequireHTMXMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	wrapped := RequireHTMXMiddleware(handler)

	t.Run("allows HTMX requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/partial", nil)
		req.Header.Set("HX-Request", "true")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "OK", rec.Body.String())
	})

	t.Run("blocks non-HTMX requests", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/partial", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("blocks wrong HX-Request value", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/partial", nil)
		req.Header.Set("HX-Request", "false")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestLimitBodyMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to read the body
		buf := make([]byte, 2<<20) // 2MB buffer
		_, err := r.Body.Read(buf)
		if err != nil && err.Error() != "EOF" {
			http.Error(w, "body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	wrapped := LimitBodyMiddleware(handler)

	t.Run("allows small JSON body", func(t *testing.T) {
		body := strings.NewReader(`{"name": "test"}`)
		req := httptest.NewRequest(http.MethodPost, "/api/test", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("allows small form body", func(t *testing.T) {
		body := strings.NewReader("name=test&value=123")
		req := httptest.NewRequest(http.MethodPost, "/api/test", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("handles nil body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrapped.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single IP",
			xff:        "203.0.113.50",
			remoteAddr: "127.0.0.1:1234",
			expected:   "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			xff:        "203.0.113.50, 70.41.3.18, 150.172.238.178",
			remoteAddr: "127.0.0.1:1234",
			expected:   "203.0.113.50",
		},
		{
			name:       "X-Forwarded-For with whitespace",
			xff:        "  203.0.113.50  ",
			remoteAddr: "127.0.0.1:1234",
			expected:   "203.0.113.50",
		},
		{
			name:       "X-Real-IP",
			xri:        "198.51.100.178",
			remoteAddr: "127.0.0.1:1234",
			expected:   "198.51.100.178",
		},
		{
			name:       "X-Real-IP with whitespace",
			xri:        "  198.51.100.178  ",
			remoteAddr: "127.0.0.1:1234",
			expected:   "198.51.100.178",
		},
		{
			name:       "X-Forwarded-For takes precedence over X-Real-IP",
			xff:        "203.0.113.50",
			xri:        "198.51.100.178",
			remoteAddr: "127.0.0.1:1234",
			expected:   "203.0.113.50",
		},
		{
			name:       "fallback to RemoteAddr with port",
			remoteAddr: "192.168.1.1:8080",
			expected:   "192.168.1.1",
		},
		{
			name:       "fallback to RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			got := GetClientIP(req)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestGenerateNonce(t *testing.T) {
	t.Run("generates base64 string", func(t *testing.T) {
		nonce, err := generateNonce()
		require.NoError(t, err)
		assert.NotEmpty(t, nonce)
		// Base64 of 16 bytes = 24 chars
		assert.Len(t, nonce, 24)
	})

	t.Run("generates unique values", func(t *testing.T) {
		n1, err := generateNonce()
		require.NoError(t, err)
		n2, err := generateNonce()
		require.NoError(t, err)
		assert.NotEqual(t, n1, n2)
	})
}
