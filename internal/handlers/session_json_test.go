package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
	"github.com/stretchr/testify/require"
)

func TestHandleSessionJSONAnonymous(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	rec := httptest.NewRecorder()

	h.HandleSessionJSON(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	var response SessionResponseJSON
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(t, SessionResponseJSON{TemperatureUnit: "recorded"}, response)
}

func TestHandleSessionStatusJSONAuthenticated(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/session/status", nil)
	req = req.WithContext(atpmiddleware.ContextWithAuth(req.Context(), "did:plc:test123456789", "sess-1"))
	rec := httptest.NewRecorder()

	h.HandleSessionStatusJSON(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	var status SessionStatusResponseJSON
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&status))
	require.Equal(t, SessionStatusResponseJSON{IsAuthenticated: true, SessionExpired: false}, status)
}

func TestHandleSessionStatusJSONAnonymousNoCookies(t *testing.T) {
	// With no OAuth configured and no cookies, an anonymous request is simply
	// not authenticated (not expired): the SPA treats this as logged-out.
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/session/status", nil)
	rec := httptest.NewRecorder()

	h.HandleSessionStatusJSON(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var status SessionStatusResponseJSON
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&status))
	require.Equal(t, SessionStatusResponseJSON{IsAuthenticated: false, SessionExpired: false}, status)
}

func TestHandleSessionStatusJSONExpiredCookiesWithoutOAuth(t *testing.T) {
	// Cookies are present (user was logged in) but no OAuth is wired on this
	// handler. We must not claim the session is expired when we cannot verify
	// it; report not-authenticated without a false-positive expiry.
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/session/status", nil)
	req.AddCookie(&http.Cookie{Name: "account_did", Value: "did:plc:test123456789"})
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "sess-1"})
	rec := httptest.NewRecorder()

	h.HandleSessionStatusJSON(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var status SessionStatusResponseJSON
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&status))
	require.Equal(t, SessionStatusResponseJSON{IsAuthenticated: false, SessionExpired: false}, status)
}
