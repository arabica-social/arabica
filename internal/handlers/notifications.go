package handlers

import (
	"fmt"
	"net/http"
	"strings"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/web/pages"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// HandleNotifications renders the notifications page
func (h *Handler) HandleNotifications(w http.ResponseWriter, r *http.Request) {
	layoutData, didStr, isAuthenticated := h.LayoutDataFromRequest(r, "Notifications")
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
				Link:         resolveNotificationLink(h.app, notif.SubjectURI),
				ActionText:   notifActionText(h.app, notif),
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
	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
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
// Format: at://did:plc:xxx/social.arabica.alpha.brew/rkey -> /brews/did:plc:xxx/rkey
func resolveNotificationLink(app *domain.App, subjectURI string) string {
	did, collection, rkey, ok := parseNotificationSubjectURI(subjectURI)
	if !ok || app == nil {
		return ""
	}

	if desc := app.DescriptorByNSID(collection); desc != nil && desc.URLPath != "" {
		return fmt.Sprintf("/%s/%s/%s", desc.URLPath, did, rkey)
	}
	return ""
}

// resolveNotificationEntityName returns the display name for the entity in a SubjectURI.
func resolveNotificationEntityName(app *domain.App, subjectURI string) string {
	_, collection, _, ok := parseNotificationSubjectURI(subjectURI)
	if !ok || app == nil {
		return "content"
	}
	if desc := app.DescriptorByNSID(collection); desc != nil {
		if desc.Noun != "" {
			return desc.Noun
		}
		if desc.DisplayName != "" {
			return strings.ToLower(desc.DisplayName)
		}
	}
	return "content"
}

func parseNotificationSubjectURI(subjectURI string) (did, collection, rkey string, ok bool) {
	if !strings.HasPrefix(subjectURI, "at://") {
		return "", "", "", false
	}

	rest := subjectURI[5:] // strip "at://"
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

// notifActionText returns human-readable action text for a notification.
func notifActionText(app *domain.App, notif arabica.Notification) string {
	entity := resolveNotificationEntityName(app, notif.SubjectURI)
	switch notif.Type {
	case arabica.NotificationLike:
		return "liked your " + entity
	case arabica.NotificationComment:
		return "commented on your " + entity
	case arabica.NotificationCommentReply:
		return "replied to your comment"
	default:
		return "interacted with your " + entity
	}
}
