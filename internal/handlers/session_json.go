package handlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/web/spa"
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
