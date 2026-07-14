package handlers

import (
	"errors"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"tangled.org/arabica.social/arabica/internal/web/spa"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

// SessionResponseJSON is the client-readable equivalent of the session data
// injected into the production SPA shell. Vite serves app.html itself in
// development, so it cannot receive those server-injected body attributes.
type SessionResponseJSON struct {
	DID                 string `json:"did"`
	Handle              string `json:"handle"`
	DisplayName         string `json:"display_name"`
	Avatar              string `json:"avatar"`
	IsAuthenticated     bool   `json:"is_authenticated"`
	IsModerator         bool   `json:"is_moderator"`
	UnreadNotifications int    `json:"unread_notifications"`
	TemperatureUnit     string `json:"temperature_unit"`
	App                 string `json:"app"`
}

// HandleSessionJSON returns the current SPA session state. It is intentionally
// public: anonymous clients receive an explicit unauthenticated response.
func (h *Handler) HandleSessionJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	did, authenticated := atpmiddleware.GetDID(r.Context())
	data := spa.SessionData{}
	if authenticated {
		data = h.ResolveSessionData(r.Context(), did)
	}
	temperatureUnit := data.TemperatureUnit
	if temperatureUnit == "" {
		temperatureUnit = "recorded"
	}

	WriteJSON(w, SessionResponseJSON{
		DID:                 did,
		Handle:              data.Handle,
		DisplayName:         data.DisplayName,
		Avatar:              data.Avatar,
		IsAuthenticated:     authenticated,
		IsModerator:         data.IsModerator,
		UnreadNotifications: data.UnreadNotificationCount,
		TemperatureUnit:     temperatureUnit,
		App:                 appName(h.app),
	}, "session")
}

// SessionStatusResponseJSON reports whether the authenticated session is still
// resumable. The SPA polls this proactively (e.g. when opening the brew form)
// so it can prompt re-authentication before a failed save instead of after.
//
// This is a best-effort local check: OAuthApp.SessionScopes loads the session
// from the local store and returns atp.ErrSessionExpired when it is gone. That
// catches sessions deleted locally (logout elsewhere, store cleared, cookies
// pointing at a missing row) but NOT a refresh token revoked server-side —
// that only surfaces lazily on the next PDS operation when token refresh fails.
// The reactive 401 path remains the safety net for the latter.
type SessionStatusResponseJSON struct {
	IsAuthenticated bool `json:"is_authenticated"`
	// SessionExpired is true when auth cookies are present but the session can
	// no longer be resumed locally. The SPA uses this to proactively open the
	// re-authentication modal.
	SessionExpired bool `json:"session_expired"`
}

// HandleSessionStatusJSON reports proactive session validity for SPA forms.
func (h *Handler) HandleSessionStatusJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	did, authenticated := atpmiddleware.GetDID(r.Context())

	// Authenticated here means the cookie-auth middleware found a resumable
	// session on this request, so the session is healthy.
	if authenticated && did != "" {
		WriteJSON(w, SessionStatusResponseJSON{IsAuthenticated: true}, "session status")
		return
	}

	// No context DID means either the user is anonymous, or cookies exist but
	// the middleware couldn't resume the session. Distinguish the two by
	// checking whether cookies are present at all: if the DID cookie is set
	// but the session wasn't loaded, the session has expired locally.
	sessionExpired := false
	if h.oauth != nil {
		didCookieName, sessCookieName := h.cookieNames()
		if didCookie, err := r.Cookie(didCookieName); err == nil && didCookie.Value != "" {
			if sessionCookie, err := r.Cookie(sessCookieName); err == nil && sessionCookie.Value != "" {
				sessionExpired = h.sessionIsExpired(r, didCookie.Value, sessionCookie.Value)
			}
		}
	}

	WriteJSON(w, SessionStatusResponseJSON{
		IsAuthenticated: false,
		SessionExpired:  sessionExpired,
	}, "session status")
}

// sessionIsExpired reports whether the session referenced by the given cookies
// can no longer be resumed. A non-error SessionScopes call means the session
// row still exists locally; any error (including atp.ErrSessionExpired) means
// the session is gone and the user should re-authenticate.
func (h *Handler) sessionIsExpired(r *http.Request, didStr, sessionID string) bool {
	if h.oauth == nil {
		return false
	}
	did, err := syntax.ParseDID(didStr)
	if err != nil {
		return true
	}
	if _, err := h.oauth.SessionScopes(r.Context(), did, sessionID); err != nil {
		return errors.Is(err, atp.ErrSessionExpired)
	}
	return false
}
