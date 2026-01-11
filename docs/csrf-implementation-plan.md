# CSRF Protection Implementation Plan

This document provides a comprehensive plan for implementing CSRF (Cross-Site Request Forgery) protection in Arabica.

## Table of Contents

1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Implementation Steps](#implementation-steps)
4. [Files to Modify](#files-to-modify)
5. [Testing Plan](#testing-plan)
6. [Rollback Plan](#rollback-plan)

---

## Overview

### What We're Protecting

All state-changing endpoints (POST, PUT, DELETE) that modify user data:

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/auth/login` | POST | User login |
| `/logout` | POST | User logout |
| `/brews` | POST | Create brew |
| `/brews/{id}` | PUT | Update brew |
| `/brews/{id}` | DELETE | Delete brew |
| `/api/beans` | POST | Create bean |
| `/api/beans/{id}` | PUT | Update bean |
| `/api/beans/{id}` | DELETE | Delete bean |
| `/api/roasters` | POST | Create roaster |
| `/api/roasters/{id}` | PUT | Update roaster |
| `/api/roasters/{id}` | DELETE | Delete roaster |
| `/api/grinders` | POST | Create grinder |
| `/api/grinders/{id}` | PUT | Update grinder |
| `/api/grinders/{id}` | DELETE | Delete grinder |
| `/api/brewers` | POST | Create brewer |
| `/api/brewers/{id}` | PUT | Update brewer |
| `/api/brewers/{id}` | DELETE | Delete brewer |

### Exempt Endpoints

| Endpoint | Reason |
|----------|--------|
| `GET /oauth/callback` | Uses OAuth `state` parameter for CSRF protection |
| `GET /*` | Read-only, no state changes |

---

## Architecture

### Token Strategy: Double Submit Cookie

We'll use the **Double Submit Cookie** pattern because:
1. Stateless - No server-side token storage needed
2. Works well with HTMX and JavaScript fetch calls
3. Simple to implement

**How it works:**

```
1. Server generates random CSRF token
2. Token sent to browser in TWO ways:
   a) As a cookie: `csrf_token=abc123` (HttpOnly=false, so JS can read it)
   b) Embedded in page (meta tag or JS variable)

3. Browser must send token back in TWO ways:
   a) Automatically via cookie (browser does this)
   b) Manually via header `X-CSRF-Token: abc123` or form field

4. Server validates: cookie token == header/form token
```

**Why this works:**
- Attacker on `evil.com` can trigger requests with cookies (browser sends automatically)
- But attacker CANNOT read the cookie value (Same-Origin Policy)
- So attacker cannot include matching header/form value
- Request is rejected

### Token Lifecycle

```
┌─────────────────────────────────────────────────────────────────┐
│                        Token Generation                          │
├─────────────────────────────────────────────────────────────────┤
│ When: First request from a browser (no existing CSRF cookie)    │
│ How:  Generate 32-byte random token using crypto/rand           │
│ Set:  Cookie `csrf_token` with value                            │
│       - Path: /                                                  │
│       - HttpOnly: false (JS needs to read it)                   │
│       - Secure: true (in production)                            │
│       - SameSite: Strict                                        │
│       - MaxAge: 86400 (24 hours, shorter than session)          │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                        Token Validation                          │
├─────────────────────────────────────────────────────────────────┤
│ When: Any POST, PUT, DELETE, PATCH request                      │
│ Steps:                                                          │
│   1. Read token from cookie                                     │
│   2. Read token from header (X-CSRF-Token) OR form (csrf_token) │
│   3. Compare using constant-time comparison                     │
│   4. If mismatch or missing: HTTP 403 Forbidden                 │
│   5. If match: Continue to handler                              │
└─────────────────────────────────────────────────────────────────┘
```

---

## Implementation Steps

### Phase 1: Backend Middleware (Go)

#### 1.1 Create CSRF Middleware

**File:** `internal/middleware/csrf.go`

```go
package middleware

import (
    "crypto/rand"
    "crypto/subtle"
    "encoding/base64"
    "net/http"
    "strings"
)

const (
    CSRFTokenCookieName = "csrf_token"
    CSRFTokenHeaderName = "X-CSRF-Token"
    CSRFTokenFormField  = "csrf_token"
    CSRFTokenLength     = 32
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

// generateToken creates a cryptographically secure random token
func generateToken() (string, error) {
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
                token, err = generateToken()
                if err != nil {
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
            w.Header().Set("X-CSRF-Token", token)
            
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
                http.Error(w, "CSRF token missing", http.StatusForbidden)
                return
            }
            
            // Constant-time comparison to prevent timing attacks
            if subtle.ConstantTimeCompare([]byte(token), []byte(submittedToken)) != 1 {
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
```

#### 1.2 Update Routing

**File:** `internal/routing/routing.go`

Add CSRF middleware to the chain:

```go
// Apply middleware in order (outermost first, innermost last)
var handler http.Handler = mux

// 1. Limit request body size (innermost - runs first on request)
handler = middleware.LimitBodyMiddleware(handler)

// 2. Apply CSRF protection (before auth, validates tokens)
csrfConfig := &middleware.CSRFConfig{
    SecureCookie:  cfg.SecureCookies, // Pass from config
    ExemptPaths:   []string{"/oauth/callback"},
}
handler = middleware.CSRFMiddleware(csrfConfig)(handler)

// 3. Apply OAuth middleware to add auth context
handler = cfg.OAuthManager.AuthMiddleware(handler)

// ... rest of middleware
```

#### 1.3 Update BFF/Render to Include Token

**File:** `internal/bff/render.go`

Add CSRF token to PageData:

```go
type PageData struct {
    Title           string
    IsAuthenticated bool
    UserDID         string
    CSRFToken       string  // ADD THIS
    // ... other fields
}

// Update all Render* functions to accept and include CSRF token
func RenderHome(w http.ResponseWriter, isAuthenticated bool, userDID string, feedItems []*feed.FeedItem, csrfToken string) error {
    data := &HomePageData{
        PageData: PageData{
            Title:           "Arabica - Coffee Brew Tracker",
            IsAuthenticated: isAuthenticated,
            UserDID:         userDID,
            CSRFToken:       csrfToken,
        },
        // ...
    }
    // ...
}
```

#### 1.4 Update Handlers to Pass Token

**File:** `internal/handlers/handlers.go`

Update handlers to pass CSRF token to templates:

```go
func (h *Handler) HandleHome(w http.ResponseWriter, r *http.Request) {
    csrfToken := middleware.GetCSRFToken(r)
    // ... pass csrfToken to render function
}
```

### Phase 2: Frontend Changes

#### 2.1 Update Templates with Hidden Fields

**File:** `templates/home.tmpl`

Add CSRF token to login form:

```html
<form method="POST" action="/auth/login" class="max-w-md mx-auto">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
    <!-- rest of form -->
</form>
```

Add CSRF token to logout form:

```html
<form action="/logout" method="POST" class="inline-block">
    <input type="hidden" name="csrf_token" value="{{.CSRFToken}}">
    <button type="submit">Logout</button>
</form>
```

**File:** `templates/brew_form.tmpl`

Add CSRF token and configure HTMX:

```html
<form id="brew-form"
    {{if .Brew}}
        hx-put="/brews/{{.Brew.RKey}}"
    {{else}}
        hx-post="/brews"
    {{end}}
    hx-headers='{"X-CSRF-Token": "{{.CSRFToken}}"}'
    hx-target="body"
    hx-swap="none">
    
    <!-- form fields -->
</form>
```

**File:** `templates/partials/brew_list_content.tmpl`

Add CSRF header to delete button:

```html
<button hx-delete="/brews/{{.RKey}}"
        hx-headers='{"X-CSRF-Token": "{{$.CSRFToken}}"}'
        hx-confirm="Are you sure?"
        hx-target="closest .brew-card"
        hx-swap="outerHTML">
    Delete
</button>
```

#### 2.2 Update JavaScript Files

**File:** `web/static/js/csrf.js` (NEW)

Create helper for CSRF token:

```javascript
/**
 * CSRF Token Helper
 * 
 * Usage:
 *   import { getCSRFToken, csrfFetch } from './csrf.js';
 *   
 *   // Manual header
 *   fetch('/api/beans', {
 *       method: 'POST',
 *       headers: { 'X-CSRF-Token': getCSRFToken() }
 *   });
 *   
 *   // Or use wrapper
 *   csrfFetch('/api/beans', { method: 'POST', body: data });
 */

/**
 * Get CSRF token from cookie
 */
export function getCSRFToken() {
    const name = 'csrf_token=';
    const decodedCookie = decodeURIComponent(document.cookie);
    const cookies = decodedCookie.split(';');
    
    for (let cookie of cookies) {
        cookie = cookie.trim();
        if (cookie.indexOf(name) === 0) {
            return cookie.substring(name.length);
        }
    }
    return '';
}

/**
 * Fetch wrapper that automatically includes CSRF token
 */
export async function csrfFetch(url, options = {}) {
    const headers = options.headers || {};
    
    // Only add CSRF token for state-changing methods
    const method = (options.method || 'GET').toUpperCase();
    if (!['GET', 'HEAD', 'OPTIONS', 'TRACE'].includes(method)) {
        headers['X-CSRF-Token'] = getCSRFToken();
    }
    
    return fetch(url, { ...options, headers });
}

// Configure HTMX to include CSRF token on all requests
document.addEventListener('DOMContentLoaded', function() {
    document.body.addEventListener('htmx:configRequest', function(event) {
        // Add CSRF token header to all HTMX requests
        event.detail.headers['X-CSRF-Token'] = getCSRFToken();
    });
});
```

**File:** `web/static/js/manage-page.js`

Update all fetch calls:

```javascript
// At top of file
import { getCSRFToken } from './csrf.js';

// Update all fetch calls to include header
async function saveBean(data, rkey = null) {
    const url = rkey ? `/api/beans/${rkey}` : '/api/beans';
    const method = rkey ? 'PUT' : 'POST';
    
    const response = await fetch(url, {
        method: method,
        headers: {
            'Content-Type': 'application/json',
            'X-CSRF-Token': getCSRFToken()  // ADD THIS
        },
        body: JSON.stringify(data)
    });
    // ...
}

async function deleteBean(rkey) {
    const response = await fetch(`/api/beans/${rkey}`, {
        method: 'DELETE',
        headers: {
            'X-CSRF-Token': getCSRFToken()  // ADD THIS
        }
    });
    // ...
}

// Apply same pattern to all other CRUD functions:
// - saveRoaster, deleteRoaster
// - saveGrinder, deleteGrinder
// - saveBrewer, deleteBrewer
```

**File:** `web/static/js/brew-form.js`

Update inline creation fetch calls:

```javascript
import { getCSRFToken } from './csrf.js';

// Update bean creation
async function createInlineBean() {
    const response = await fetch('/api/beans', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-CSRF-Token': getCSRFToken()  // ADD THIS
        },
        body: JSON.stringify(beanData)
    });
    // ...
}

// Apply to grinder and brewer creation too
```

#### 2.3 Update Layout Template

**File:** `templates/layout.tmpl`

Include CSRF script globally:

```html
<head>
    <!-- ... -->
    <script type="module" src="/static/js/csrf.js"></script>
</head>
```

Or add meta tag for non-module scripts:

```html
<head>
    <meta name="csrf-token" content="{{.CSRFToken}}">
</head>
```

---

## Files to Modify

### New Files

| File | Purpose |
|------|---------|
| `internal/middleware/csrf.go` | CSRF middleware implementation |
| `internal/middleware/csrf_test.go` | Unit tests for CSRF middleware |
| `web/static/js/csrf.js` | JavaScript CSRF helper |
| `test/csrf/csrf_test.go` | Integration tests |
| `test/csrf/server_test.go` | Test server setup |

### Modified Files

| File | Changes |
|------|---------|
| `internal/routing/routing.go` | Add CSRF middleware to chain |
| `internal/bff/render.go` | Add CSRFToken to PageData and all render functions |
| `internal/handlers/handlers.go` | Pass CSRF token to render functions |
| `internal/handlers/auth.go` | Pass CSRF token to home render |
| `templates/layout.tmpl` | Include CSRF meta tag or script |
| `templates/home.tmpl` | Add hidden fields to login/logout forms |
| `templates/brew_form.tmpl` | Add HTMX headers with CSRF token |
| `templates/partials/brew_list_content.tmpl` | Add CSRF header to delete buttons |
| `templates/partials/manage_content.tmpl` | Ensure CSRF token available |
| `web/static/js/manage-page.js` | Add CSRF header to all fetch calls |
| `web/static/js/brew-form.js` | Add CSRF header to inline creation |
| `web/static/js/data-cache.js` | Check if any POST calls need CSRF |

---

## Testing Plan

### Unit Tests

**File:** `internal/middleware/csrf_test.go`

```go
package middleware

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestCSRFTokenGeneration(t *testing.T) {
    // Test that tokens are generated correctly
    token, err := generateToken()
    if err != nil {
        t.Fatalf("Failed to generate token: %v", err)
    }
    if len(token) == 0 {
        t.Error("Generated token is empty")
    }
    
    // Test uniqueness
    token2, _ := generateToken()
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
        ExemptPaths: []string{"/oauth/callback"},
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
```

### Integration Tests

**File:** `test/csrf/main_test.go`

These tests run against a real server instance:

```go
//go:build integration

package csrf_test

import (
    "encoding/json"
    "io"
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "os"
    "strings"
    "testing"
    "time"
)

var serverURL string

func TestMain(m *testing.M) {
    // Get server URL from environment or use default test port
    serverURL = os.Getenv("TEST_SERVER_URL")
    if serverURL == "" {
        serverURL = "http://localhost:18911"
    }
    
    // Wait for server to be ready
    client := &http.Client{Timeout: 1 * time.Second}
    for i := 0; i < 30; i++ {
        resp, err := client.Get(serverURL + "/")
        if err == nil {
            resp.Body.Close()
            break
        }
        time.Sleep(100 * time.Millisecond)
    }
    
    os.Exit(m.Run())
}

// Helper to create client with cookie jar
func newClient() *http.Client {
    jar, _ := cookiejar.New(nil)
    return &http.Client{
        Jar:     jar,
        Timeout: 10 * time.Second,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse // Don't follow redirects
        },
    }
}

// Helper to get CSRF token from cookie
func getCSRFToken(client *http.Client, serverURL string) (string, error) {
    u, _ := url.Parse(serverURL)
    for _, cookie := range client.Jar.Cookies(u) {
        if cookie.Name == "csrf_token" {
            return cookie.Value, nil
        }
    }
    return "", nil
}

func TestCSRF_HomePageSetsToken(t *testing.T) {
    client := newClient()
    
    resp, err := client.Get(serverURL + "/")
    if err != nil {
        t.Fatalf("Failed to get home page: %v", err)
    }
    defer resp.Body.Close()
    
    token, _ := getCSRFToken(client, serverURL)
    if token == "" {
        t.Error("CSRF token cookie not set on home page")
    }
}

func TestCSRF_LoginWithoutToken_Fails(t *testing.T) {
    client := newClient()
    
    // First visit to get cookies
    client.Get(serverURL + "/")
    
    // Try login without CSRF token
    form := url.Values{}
    form.Set("handle", "test.bsky.social")
    
    req, _ := http.NewRequest("POST", serverURL+"/auth/login", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    // Clear the CSRF token header (simulating attack)
    resp, err := client.Do(req)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()
    
    // Without token, should get 403
    if resp.StatusCode != http.StatusForbidden {
        body, _ := io.ReadAll(resp.Body)
        t.Errorf("Expected 403 Forbidden, got %d: %s", resp.StatusCode, string(body))
    }
}

func TestCSRF_LoginWithToken_Succeeds(t *testing.T) {
    client := newClient()
    
    // First visit to get CSRF cookie
    client.Get(serverURL + "/")
    
    token, _ := getCSRFToken(client, serverURL)
    if token == "" {
        t.Fatal("No CSRF token cookie")
    }
    
    // Submit login with CSRF token
    form := url.Values{}
    form.Set("handle", "test.bsky.social")
    form.Set("csrf_token", token)
    
    req, _ := http.NewRequest("POST", serverURL+"/auth/login", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    resp, err := client.Do(req)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()
    
    // Should redirect (302) to OAuth, not 403
    if resp.StatusCode == http.StatusForbidden {
        body, _ := io.ReadAll(resp.Body)
        t.Errorf("CSRF validation failed with valid token: %s", string(body))
    }
}

func TestCSRF_APIDeleteWithoutToken_Fails(t *testing.T) {
    client := newClient()
    
    // First visit to get cookies
    client.Get(serverURL + "/")
    
    // Try DELETE without CSRF token
    req, _ := http.NewRequest("DELETE", serverURL+"/api/beans/test123", nil)
    
    resp, err := client.Do(req)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusForbidden {
        t.Errorf("Expected 403, got %d", resp.StatusCode)
    }
}

func TestCSRF_APIDeleteWithToken_PassesValidation(t *testing.T) {
    client := newClient()
    
    // First visit to get CSRF cookie
    client.Get(serverURL + "/")
    
    token, _ := getCSRFToken(client, serverURL)
    
    // Try DELETE with CSRF token
    req, _ := http.NewRequest("DELETE", serverURL+"/api/beans/test123", nil)
    req.Header.Set("X-CSRF-Token", token)
    
    resp, err := client.Do(req)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()
    
    // Should NOT be 403 (might be 401 for auth, or 404 for not found, but not 403 CSRF)
    if resp.StatusCode == http.StatusForbidden {
        body, _ := io.ReadAll(resp.Body)
        if strings.Contains(string(body), "CSRF") {
            t.Errorf("CSRF validation failed with valid token")
        }
    }
}

func TestCSRF_APIPOSTWithToken_PassesValidation(t *testing.T) {
    client := newClient()
    
    // First visit to get CSRF cookie
    client.Get(serverURL + "/")
    
    token, _ := getCSRFToken(client, serverURL)
    
    // Try POST with CSRF token
    body, _ := json.Marshal(map[string]string{"name": "Test Bean"})
    req, _ := http.NewRequest("POST", serverURL+"/api/beans", strings.NewReader(string(body)))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-CSRF-Token", token)
    
    resp, err := client.Do(req)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()
    
    // Should NOT be 403 CSRF error (might be 401 for auth)
    if resp.StatusCode == http.StatusForbidden {
        respBody, _ := io.ReadAll(resp.Body)
        if strings.Contains(string(respBody), "CSRF") {
            t.Errorf("CSRF validation failed with valid token")
        }
    }
}

func TestCSRF_CrossOriginAttack_Fails(t *testing.T) {
    // Simulate cross-origin attack: attacker has no way to get CSRF token
    client := &http.Client{Timeout: 10 * time.Second}
    
    // Attack request without any cookies or tokens
    form := url.Values{}
    form.Set("handle", "victim.bsky.social")
    
    req, _ := http.NewRequest("POST", serverURL+"/auth/login", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    // No CSRF token - simulating cross-origin request
    
    resp, err := client.Do(req)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusForbidden {
        t.Errorf("Cross-origin attack should be blocked, got status %d", resp.StatusCode)
    }
}

func TestCSRF_TokenReuse_Works(t *testing.T) {
    client := newClient()
    
    // Get token
    client.Get(serverURL + "/")
    token, _ := getCSRFToken(client, serverURL)
    
    // Use token multiple times (should work until it expires)
    for i := 0; i < 3; i++ {
        req, _ := http.NewRequest("DELETE", serverURL+"/api/beans/test"+string(rune(i)), nil)
        req.Header.Set("X-CSRF-Token", token)
        
        resp, err := client.Do(req)
        if err != nil {
            t.Fatalf("Request %d failed: %v", i, err)
        }
        resp.Body.Close()
        
        if resp.StatusCode == http.StatusForbidden {
            body, _ := io.ReadAll(resp.Body)
            if strings.Contains(string(body), "CSRF") {
                t.Errorf("Token should be reusable, failed on request %d", i)
            }
        }
    }
}

func TestCSRF_MismatchedTokens_Fails(t *testing.T) {
    client := newClient()
    
    // Get legitimate token
    client.Get(serverURL + "/")
    token, _ := getCSRFToken(client, serverURL)
    
    // Send request with different token in header vs cookie
    req, _ := http.NewRequest("DELETE", serverURL+"/api/beans/test123", nil)
    req.Header.Set("X-CSRF-Token", "completely-different-token")
    // Cookie still has real token (from jar)
    
    resp, err := client.Do(req)
    if err != nil {
        t.Fatalf("Request failed: %v", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusForbidden {
        t.Errorf("Mismatched tokens should return 403, got %d (cookie: %s)", resp.StatusCode, token)
    }
}
```

### Running Tests

**File:** `test/csrf/README.md`

```markdown
# CSRF Integration Tests

These tests verify CSRF protection works correctly against a running server.

## Prerequisites

1. Build the server
2. Have the CSRF middleware implemented

## Running Tests

### Option 1: Start server manually

```bash
# Terminal 1: Start test server on different port
PORT=18911 go run cmd/server/main.go

# Terminal 2: Run tests
TEST_SERVER_URL=http://localhost:18911 go test -tags=integration ./test/csrf/...
```

### Option 2: Use test script

```bash
./test/csrf/run_tests.sh
```

## Test Coverage

| Test | Validates |
|------|-----------|
| `TestCSRF_HomePageSetsToken` | Token cookie is set on first visit |
| `TestCSRF_LoginWithoutToken_Fails` | Form POST without token blocked |
| `TestCSRF_LoginWithToken_Succeeds` | Form POST with token allowed |
| `TestCSRF_APIDeleteWithoutToken_Fails` | API DELETE without token blocked |
| `TestCSRF_APIDeleteWithToken_PassesValidation` | API DELETE with token allowed |
| `TestCSRF_APIPOSTWithToken_PassesValidation` | API POST with token allowed |
| `TestCSRF_CrossOriginAttack_Fails` | Attack without cookies blocked |
| `TestCSRF_TokenReuse_Works` | Token can be used multiple times |
| `TestCSRF_MismatchedTokens_Fails` | Wrong token in header blocked |
```

**File:** `test/csrf/run_tests.sh`

```bash
#!/bin/bash

# CSRF Integration Test Runner
# Starts a test server, runs tests, then cleans up

set -e

PORT=18911
SERVER_PID=""
TEST_DB="/tmp/arabica-csrf-test.db"

cleanup() {
    if [ -n "$SERVER_PID" ]; then
        echo "Stopping test server (PID: $SERVER_PID)..."
        kill $SERVER_PID 2>/dev/null || true
        wait $SERVER_PID 2>/dev/null || true
    fi
    rm -f "$TEST_DB"
}

trap cleanup EXIT

echo "=== CSRF Integration Tests ==="
echo ""

# Build server
echo "Building server..."
go build -o /tmp/arabica-csrf-test ./cmd/server/main.go

# Start server
echo "Starting test server on port $PORT..."
ARABICA_DB_PATH="$TEST_DB" PORT=$PORT /tmp/arabica-csrf-test &
SERVER_PID=$!

# Wait for server to be ready
echo "Waiting for server to start..."
for i in $(seq 1 30); do
    if curl -s "http://localhost:$PORT/" > /dev/null 2>&1; then
        echo "Server ready!"
        break
    fi
    sleep 0.1
done

# Run tests
echo ""
echo "Running tests..."
TEST_SERVER_URL="http://localhost:$PORT" go test -v -tags=integration ./test/csrf/...

echo ""
echo "=== Tests Complete ==="
```

---

## Rollback Plan

If CSRF protection causes issues:

### Quick Disable

1. Comment out CSRF middleware in `internal/routing/routing.go`
2. Rebuild and deploy

### Gradual Rollout

1. Start with `ExemptPaths: []string{"/*"}` (exempt all paths)
2. Remove exemptions one endpoint at a time
3. Monitor for 403 errors in logs

### Feature Flag

Add environment variable:

```go
// In routing.go
if os.Getenv("ENABLE_CSRF") == "true" {
    handler = middleware.CSRFMiddleware(csrfConfig)(handler)
}
```

---

## Monitoring

### Metrics to Watch

1. **403 Forbidden rate** - Sudden spike indicates broken CSRF
2. **Login success rate** - Drop indicates form not sending token
3. **API error rate** - Increase indicates JS not sending headers

### Log Messages

The middleware should log:
- Token generation (DEBUG level)
- Validation failures (WARN level) with client IP, path, method

```go
log.Warn().
    Str("client_ip", getClientIP(r)).
    Str("path", r.URL.Path).
    Str("method", r.Method).
    Msg("CSRF validation failed")
```

---

## Timeline

| Day | Task |
|-----|------|
| 1 | Implement middleware, unit tests |
| 2 | Update templates, test forms |
| 3 | Update JavaScript, test fetch calls |
| 4 | Integration tests, manual QA |
| 5 | Deploy to staging, monitor |
| 6 | Deploy to production |

---

## Checklist

### Backend
- [ ] Create `internal/middleware/csrf.go`
- [ ] Add unit tests
- [ ] Update routing to include middleware
- [ ] Update PageData with CSRFToken
- [ ] Update all render functions
- [ ] Update all handlers

### Frontend
- [ ] Create `web/static/js/csrf.js`
- [ ] Update `templates/layout.tmpl`
- [ ] Update `templates/home.tmpl` (login/logout forms)
- [ ] Update `templates/brew_form.tmpl`
- [ ] Update `templates/partials/brew_list_content.tmpl`
- [ ] Update `web/static/js/manage-page.js`
- [ ] Update `web/static/js/brew-form.js`

### Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing: login form
- [ ] Manual testing: logout form
- [ ] Manual testing: create brew
- [ ] Manual testing: edit brew
- [ ] Manual testing: delete brew
- [ ] Manual testing: manage page CRUD
- [ ] Test cross-origin attack blocked

### Deployment
- [ ] Deploy to staging
- [ ] Verify no 403 errors
- [ ] Deploy to production
- [ ] Monitor error rates
