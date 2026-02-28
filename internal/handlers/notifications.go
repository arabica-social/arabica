package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"arabica/internal/atproto"
	"arabica/internal/web/pages"

	"github.com/rs/zerolog/log"
)

// HandleNotifications renders the notifications page
func (h *Handler) HandleNotifications(w http.ResponseWriter, r *http.Request) {
	layoutData, didStr, isAuthenticated := h.layoutDataFromRequest(r, "Notifications")
	if !isAuthenticated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	cursor := r.URL.Query().Get("cursor")

	var props pages.NotificationsProps

	if h.feedIndex != nil {
		notifications, nextCursor, err := h.feedIndex.GetNotifications(didStr, 30, cursor)
		if err != nil {
			log.Error().Err(err).Str("did", didStr).Msg("Failed to get notifications")
			http.Error(w, "Failed to load notifications", http.StatusInternalServerError)
			return
		}

		props.NextCursor = nextCursor

		// Resolve actor profiles and links for each notification
		for _, notif := range notifications {
			item := pages.NotificationItem{
				Notification: notif,
				Link:         resolveNotificationLink(notif.SubjectURI),
			}

			profile, err := h.feedIndex.GetProfile(r.Context(), notif.ActorDID)
			if err == nil && profile != nil {
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

			props.Notifications = append(props.Notifications, item)
		}

		// Mark all as read when the page is viewed
		if err := h.feedIndex.MarkAllRead(didStr); err != nil {
			log.Warn().Err(err).Str("did", didStr).Msg("Failed to mark notifications as read on view")
		}
	}

	if err := pages.Notifications(layoutData, props).Render(r.Context(), w); err != nil {
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
		log.Error().Err(err).Msg("Failed to render notifications page")
	}
}

// HandleNotificationsMarkRead marks all notifications as read
func (h *Handler) HandleNotificationsMarkRead(w http.ResponseWriter, r *http.Request) {
	didStr, err := atproto.GetAuthenticatedDID(r.Context())
	if err != nil || didStr == "" {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	if h.feedIndex != nil {
		if err := h.feedIndex.MarkAllRead(didStr); err != nil {
			log.Error().Err(err).Str("did", didStr).Msg("Failed to mark notifications as read")
			http.Error(w, "Failed to mark notifications as read", http.StatusInternalServerError)
			return
		}
	}

	// Redirect back to notifications page
	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}

// resolveNotificationLink converts a SubjectURI (AT-URI) to a local page URL.
// All notification types store a brew AT-URI as SubjectURI.
// Format: at://did:plc:xxx/social.arabica.alpha.brew/rkey -> /brews/rkey?owner=did:plc:xxx
func resolveNotificationLink(subjectURI string) string {
	if !strings.HasPrefix(subjectURI, "at://") {
		return ""
	}

	rest := subjectURI[5:] // strip "at://"
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 3 {
		return ""
	}

	did := parts[0]
	collection := parts[1]
	rkey := parts[2]

	switch collection {
	case atproto.NSIDBrew:
		return fmt.Sprintf("/brews/%s?owner=%s", rkey, did)
	default:
		return ""
	}
}
