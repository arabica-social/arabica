package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCSRFTokenGeneration(t *testing.T) {
	// Test that tokens are generated correctly
	token, err := generateCSRFToken()
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	if len(token) == 0 {
		t.Error("Generated token is empty")
	}

	// Test uniqueness
	token2, _ := generateCSRFToken()
	if token == token2 {
		t.Error("Tokens should be unique")
	}
}

func TestCSRFMiddleware_SetsTokenCookie(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check cookie is set
	cookies := rec.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == CSRFTokenCookieName {
			csrfCookie = c
			break
		}
	}

	if csrfCookie == nil {
		t.Error("CSRF cookie not set")
	}
	if csrfCookie.Value == "" {
		t.Error("CSRF cookie value is empty")
	}
	// Verify cookie settings
	if csrfCookie.HttpOnly {
		t.Error("CSRF cookie should not be HttpOnly (JS needs to read it)")
	}
	if csrfCookie.SameSite != http.SameSiteStrictMode {
		t.Error("CSRF cookie should have SameSite=Strict")
	}
}

func TestCSRFMiddleware_SetsResponseHeader(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check response header is set
	headerToken := rec.Header().Get(CSRFTokenHeaderName)
	if headerToken == "" {
		t.Error("CSRF token not set in response header")
	}
}

func TestCSRFMiddleware_GETRequestsExempt(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// GET without token should succeed
	req := httptest.NewRequest("GET", "/some-page", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET request should succeed, got status %d", rec.Code)
	}
}

func TestCSRFMiddleware_HEADRequestsExempt(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("HEAD", "/some-page", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("HEAD request should succeed, got status %d", rec.Code)
	}
}

func TestCSRFMiddleware_OPTIONSRequestsExempt(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("OPTIONS", "/some-page", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("OPTIONS request should succeed, got status %d", rec.Code)
	}
}

func TestCSRFMiddleware_POSTWithoutToken_Fails(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// POST without token should fail
	req := httptest.NewRequest("POST", "/api/beans", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("POST without CSRF token should return 403, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_POSTWithValidToken_Succeeds(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First, get a token via GET
	getReq := httptest.NewRequest("GET", "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == CSRFTokenCookieName {
			token = c.Value
			break
		}
	}

	if token == "" {
		t.Fatal("No CSRF token cookie was set")
	}

	// Now POST with valid token
	postReq := httptest.NewRequest("POST", "/api/beans", strings.NewReader("{}"))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set(CSRFTokenHeaderName, token)
	postReq.AddCookie(&http.Cookie{Name: CSRFTokenCookieName, Value: token})
	postRec := httptest.NewRecorder()

	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Errorf("POST with valid CSRF token should succeed, got %d", postRec.Code)
	}
}

func TestCSRFMiddleware_POSTWithInvalidToken_Fails(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First, get a token
	getReq := httptest.NewRequest("GET", "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == CSRFTokenCookieName {
			token = c.Value
			break
		}
	}

	// POST with wrong token
	postReq := httptest.NewRequest("POST", "/api/beans", strings.NewReader("{}"))
	postReq.Header.Set("Content-Type", "application/json")
	postReq.Header.Set(CSRFTokenHeaderName, "wrong-token")
	postReq.AddCookie(&http.Cookie{Name: CSRFTokenCookieName, Value: token})
	postRec := httptest.NewRecorder()

	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusForbidden {
		t.Errorf("POST with invalid CSRF token should return 403, got %d", postRec.Code)
	}
}

func TestCSRFMiddleware_ExemptPath(t *testing.T) {
	config := &CSRFConfig{
		ExemptPaths:   []string{"/oauth/callback"},
		ExemptMethods: []string{"GET", "HEAD", "OPTIONS", "TRACE"},
	}
	handler := CSRFMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// POST to exempt path without token should succeed
	req := httptest.NewRequest("POST", "/oauth/callback", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Exempt path should succeed without token, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_ExemptPathPrefix(t *testing.T) {
	config := &CSRFConfig{
		ExemptPaths:   []string{"/oauth/"},
		ExemptMethods: []string{"GET", "HEAD", "OPTIONS", "TRACE"},
	}
	handler := CSRFMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// POST to path with exempt prefix without token should succeed
	req := httptest.NewRequest("POST", "/oauth/callback?code=123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Exempt path prefix should succeed without token, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_FormField(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Get token first
	getReq := httptest.NewRequest("GET", "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == CSRFTokenCookieName {
			token = c.Value
			break
		}
	}

	// POST with form field instead of header
	formData := "csrf_token=" + token + "&name=test"
	postReq := httptest.NewRequest("POST", "/api/beans", strings.NewReader(formData))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.AddCookie(&http.Cookie{Name: CSRFTokenCookieName, Value: token})
	postRec := httptest.NewRecorder()

	handler.ServeHTTP(postRec, postReq)

	if postRec.Code != http.StatusOK {
		t.Errorf("POST with form field CSRF token should succeed, got %d", postRec.Code)
	}
}

func TestCSRFMiddleware_DELETE(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// DELETE without token should fail
	req := httptest.NewRequest("DELETE", "/api/beans/123", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("DELETE without CSRF token should return 403, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_PUT(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// PUT without token should fail
	req := httptest.NewRequest("PUT", "/api/beans/123", strings.NewReader("{}"))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("PUT without CSRF token should return 403, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_DELETE_WithValidToken(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Get token first
	getReq := httptest.NewRequest("GET", "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == CSRFTokenCookieName {
			token = c.Value
			break
		}
	}

	// DELETE with valid token
	req := httptest.NewRequest("DELETE", "/api/beans/123", nil)
	req.Header.Set(CSRFTokenHeaderName, token)
	req.AddCookie(&http.Cookie{Name: CSRFTokenCookieName, Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("DELETE with valid CSRF token should succeed, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_PUT_WithValidToken(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Get token first
	getReq := httptest.NewRequest("GET", "/", nil)
	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, getReq)

	var token string
	for _, c := range getRec.Result().Cookies() {
		if c.Name == CSRFTokenCookieName {
			token = c.Value
			break
		}
	}

	// PUT with valid token
	req := httptest.NewRequest("PUT", "/api/beans/123", strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(CSRFTokenHeaderName, token)
	req.AddCookie(&http.Cookie{Name: CSRFTokenCookieName, Value: token})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("PUT with valid CSRF token should succeed, got %d", rec.Code)
	}
}

func TestCSRFMiddleware_SecureCookie(t *testing.T) {
	config := &CSRFConfig{
		SecureCookie:  true,
		ExemptMethods: []string{"GET", "HEAD", "OPTIONS", "TRACE"},
	}
	handler := CSRFMiddleware(config)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Check cookie has Secure flag
	cookies := rec.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == CSRFTokenCookieName {
			csrfCookie = c
			break
		}
	}

	if csrfCookie == nil {
		t.Fatal("CSRF cookie not set")
	}
	if !csrfCookie.Secure {
		t.Error("CSRF cookie should have Secure flag when SecureCookie=true")
	}
}

func TestGetCSRFToken(t *testing.T) {
	// Create request with CSRF cookie
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFTokenCookieName, Value: "test-token-123"})

	token := GetCSRFToken(req)
	if token != "test-token-123" {
		t.Errorf("GetCSRFToken returned wrong value: %s", token)
	}
}

func TestGetCSRFToken_NoCookie(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)

	token := GetCSRFToken(req)
	if token != "" {
		t.Errorf("GetCSRFToken should return empty string when no cookie: %s", token)
	}
}

func TestCSRFMiddleware_ReusesExistingToken(t *testing.T) {
	handler := CSRFMiddleware(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Make request with existing token
	existingToken := "existing-token-abc123"
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: CSRFTokenCookieName, Value: existingToken})
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should not set a new cookie (only set when no token exists)
	cookies := rec.Result().Cookies()
	for _, c := range cookies {
		if c.Name == CSRFTokenCookieName {
			t.Error("Should not set new CSRF cookie when valid one already exists")
		}
	}

	// Response header should contain the existing token
	headerToken := rec.Header().Get(CSRFTokenHeaderName)
	if headerToken != existingToken {
		t.Errorf("Response header should contain existing token, got: %s", headerToken)
	}
}
