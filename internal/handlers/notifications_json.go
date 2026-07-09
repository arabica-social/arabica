package handlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/notifications"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// NotificationItemJSON is the JSON-serializable view of a notification with
// resolved actor profile and link/action text. The underlying notifications.Notification
// struct already has json tags, but the actor profile fields (handle, display_name,
// avatar) and the link/action_text are computed per-request by the handler.
type NotificationItemJSON struct {
	notifications.Notification
	ActorHandle      string `json:"actor_handle"`
	ActorDisplayName string `json:"actor_display_name,omitempty"`
	ActorAvatar      string `json:"actor_avatar,omitempty"`
	Link             string `json:"link"`
	ActionText       string `json:"action_text"`
}

// NotificationsResponseJSON is the JSON envelope returned by GET /api/notifications.
type NotificationsResponseJSON struct {
	Notifications []NotificationItemJSON `json:"notifications"`
	NextCursor    string                 `json:"next_cursor"`
}

// HandleNotificationsJSON returns the user's notifications as JSON for the
// SvelteKit SPA. Reuses the same feedIndex.GetNotifications pipeline and
// actor/link resolution as the HTML handler.
func (h *Handler) HandleNotificationsJSON(w http.ResponseWriter, r *http.Request) {
	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	cursor := r.URL.Query().Get("cursor")

	resp := NotificationsResponseJSON{
		Notifications: []NotificationItemJSON{},
	}

	if h.feedIndex != nil {
		notifs, nextCursor, err := h.feedIndex.GetNotifications(didStr, 30, cursor)
		if err != nil {
			log.Error().Err(err).Str("did", didStr).Msg("Failed to get notifications")
			WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to load notifications")
			return
		}

		resp.NextCursor = nextCursor

		for _, notif := range notifs {
			item := NotificationItemJSON{
				Notification: notif,
				Link:         resolveNotificationLink(h.app, notif.SubjectURI),
				ActionText:   notifActionText(h.app, notif),
			}

			if profile, err := h.feedIndex.GetProfile(r.Context(), notif.ActorDID); err == nil && profile != nil {
				item.ActorHandle = profile.Handle
				if profile.DisplayName != nil {
					item.ActorDisplayName = *profile.DisplayName
				}
				if profile.Avatar != nil {
					item.ActorAvatar = *profile.Avatar
				}
			} else {
				item.ActorHandle = notif.ActorDID
			}

			resp.Notifications = append(resp.Notifications, item)
		}

		// Mark all as read when the notifications are viewed
		if err := h.feedIndex.MarkAllRead(didStr); err != nil {
			log.Warn().Err(err).Str("did", didStr).Msg("Failed to mark notifications as read on view")
		}
	}

	WriteJSON(w, resp, "notifications")
}

// HandleNotificationsMarkReadJSON marks all notifications as read and returns
// a JSON confirmation for the SvelteKit SPA.
func (h *Handler) HandleNotificationsMarkReadJSON(w http.ResponseWriter, r *http.Request) {
	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		WriteJSONError(w, http.StatusUnauthorized, "authentication_required", "Authentication required")
		return
	}

	if h.feedIndex != nil {
		if err := h.feedIndex.MarkAllRead(didStr); err != nil {
			log.Error().Err(err).Str("did", didStr).Msg("Failed to mark notifications as read")
			WriteJSONError(w, http.StatusInternalServerError, "internal_error", "Failed to mark notifications as read")
			return
		}
	}

	WriteJSON(w, map[string]bool{"read": true}, "notifications-read")
}
