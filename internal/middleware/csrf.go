package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/rs/zerolog/log"
)

const (
	// CSRFTokenCookieName is the name of the cookie that stores the CSRF token
	CSRFTokenCookieName = "csrf_token"
	// CSRFTokenHeaderName is the HTTP header name for submitting the CSRF token
	CSRFTokenHeaderName = "X-CSRF-Token"
	// CSRFTokenFormField is the form field name for submitting the CSRF token
	CSRFTokenFormField = "csrf_token"
	// CSRFTokenLength is the number of random bytes used to generate the token
	CSRFTokenLength = 32
)

// CSRFConfig holds CSRF middleware configuration
type CSRFConfig struct {
	// SecureCookie sets the Secure flag on the CSRF cookie
	SecureCookie bool

	// ExemptPaths are paths that skip CSRF validation (e.g., OAuth callback)
	ExemptPaths []string

	// ExemptMethods are HTTP methods that skip CSRF validation
	// Default: GET, HEAD, OPTIONS, TRACE
	ExemptMethods []string
}

// DefaultCSRFConfig returns default configuration
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		SecureCookie:  false,
		ExemptPaths:   []string{"/oauth/callback"},
		ExemptMethods: []string{"GET", "HEAD", "OPTIONS", "TRACE"},
	}
}

// generateCSRFToken creates a cryptographically secure random token
func generateCSRFToken() (string, error) {
	bytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CSRFMiddleware provides CSRF protection using double-submit cookie pattern
func CSRFMiddleware(config *CSRFConfig) func(http.Handler) http.Handler {
	if config == nil {
		config = DefaultCSRFConfig()
	}

	// Build exempt method set for fast lookup
	exemptMethods := make(map[string]bool)
	for _, m := range config.ExemptMethods {
		exemptMethods[strings.ToUpper(m)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get or generate CSRF token
			var token string
			cookie, err := r.Cookie(CSRFTokenCookieName)
			if err == nil && cookie.Value != "" {
				token = cookie.Value
			} else {
				// Generate new token
				token, err = generateCSRFToken()
				if err != nil {
					log.Error().Err(err).Msg("Failed to generate CSRF token")
					http.Error(w, "Internal server error", http.StatusInternalServerError)
					return
				}

				// Set cookie
				http.SetCookie(w, &http.Cookie{
					Name:     CSRFTokenCookieName,
					Value:    token,
					Path:     "/",
					HttpOnly: false, // JS needs to read this
					Secure:   config.SecureCookie,
					SameSite: http.SameSiteStrictMode,
					MaxAge:   86400, // 24 hours
				})
			}

			// Store token in response header for JS to access
			// This is an alternative to reading from cookie
			w.Header().Set(CSRFTokenHeaderName, token)

			// Check if method requires validation
			if exemptMethods[r.Method] {
				next.ServeHTTP(w, r)
				return
			}

			// Check if path is exempt
			for _, path := range config.ExemptPaths {
				if r.URL.Path == path || strings.HasPrefix(r.URL.Path, path) {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Validate CSRF token
			// Try header first (JavaScript requests)
			submittedToken := r.Header.Get(CSRFTokenHeaderName)

			// Fall back to form field (traditional forms)
			if submittedToken == "" {
				submittedToken = r.FormValue(CSRFTokenFormField)
			}

			// Validate token
			if submittedToken == "" {
				log.Warn().
					Str("client_ip", getClientIP(r)).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("CSRF token missing")
				http.Error(w, "CSRF token missing", http.StatusForbidden)
				return
			}

			// Constant-time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(token), []byte(submittedToken)) != 1 {
				log.Warn().
					Str("client_ip", getClientIP(r)).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("CSRF token invalid")
				http.Error(w, "CSRF token invalid", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetCSRFToken extracts the CSRF token from request cookies
// Used by template rendering to include token in forms
func GetCSRFToken(r *http.Request) string {
	cookie, err := r.Cookie(CSRFTokenCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}
