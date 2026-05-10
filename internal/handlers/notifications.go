package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/web/pages"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

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
				ActionText:   notifActionText(notif),
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

// collectionURLPath maps AT Protocol collection NSIDs to their URL path prefixes.
var collectionURLPath = map[string]string{
	arabica.NSIDBrew:    "/brews/",
	arabica.NSIDBean:    "/beans/",
	arabica.NSIDRoaster: "/roasters/",
	arabica.NSIDGrinder: "/grinders/",
	arabica.NSIDBrewer:  "/brewers/",
	arabica.NSIDRecipe:  "/recipes/",
}

// collectionDisplayName maps AT Protocol collection NSIDs to human-readable names.
var collectionDisplayName = map[string]string{
	arabica.NSIDBrew:    "brew",
	arabica.NSIDBean:    "bean",
	arabica.NSIDRoaster: "roaster",
	arabica.NSIDGrinder: "grinder",
	arabica.NSIDBrewer:  "brewer",
	arabica.NSIDRecipe:  "recipe",
}

// resolveNotificationLink converts a SubjectURI (AT-URI) to a local page URL.
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

	if prefix, ok := collectionURLPath[collection]; ok {
		return fmt.Sprintf("%s%s?owner=%s", prefix, rkey, did)
	}
	return ""
}

// resolveNotificationEntityName returns the display name for the entity in a SubjectURI.
func resolveNotificationEntityName(subjectURI string) string {
	if !strings.HasPrefix(subjectURI, "at://") {
		return "content"
	}
	rest := subjectURI[5:]
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 3 {
		return "content"
	}
	if name, ok := collectionDisplayName[parts[1]]; ok {
		return name
	}
	return "content"
}

// notifActionText returns human-readable action text for a notification.
func notifActionText(notif arabica.Notification) string {
	entity := resolveNotificationEntityName(notif.SubjectURI)
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
