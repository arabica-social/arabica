package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/notifications"
)

// HandleNotificationsMarkRead marks all notifications as read. The SPA always
// sends Accept: application/json, so this delegates to the JSON path.
func (h *Handler) HandleNotificationsMarkRead(w http.ResponseWriter, r *http.Request) {
	h.HandleNotificationsMarkReadJSON(w, r)
}

// resolveNotificationLink converts a SubjectURI (AT-URI) to a local page URL.
// Format: at://did:plc:xxx/social.arabica.alpha.brew/rkey -> /brews/did:plc:xxx/rkey
func resolveNotificationLink(app *domain.App, subjectURI string) string {
	did, collection, rkey, ok := parseNotificationSubjectURI(subjectURI)
	if !ok || app == nil {
		return ""
	}

	if route, ok := app.EntityRouteByNSID(collection); ok && route.Path != "" {
		return fmt.Sprintf("/%s/%s/%s", route.Path, did, rkey)
	}
	return ""
}

// resolveNotificationEntityName returns the display name for the entity in a SubjectURI.
func resolveNotificationEntityName(app *domain.App, subjectURI string) string {
	_, collection, _, ok := parseNotificationSubjectURI(subjectURI)
	if !ok || app == nil {
		return "content"
	}
	if route, ok := app.EntityRouteByNSID(collection); ok && route.Noun != "" {
		return route.Noun
	}
	if desc := app.DescriptorByNSID(collection); desc != nil && desc.DisplayName != "" {
		return strings.ToLower(desc.DisplayName)
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
func notifActionText(app *domain.App, notif notifications.Notification) string {
	entity := resolveNotificationEntityName(app, notif.SubjectURI)
	switch notif.Type {
	case notifications.Like:
		return "liked your " + entity
	case notifications.Comment:
		return "commented on your " + entity
	case notifications.CommentReply:
		return "replied to your comment"
	default:
		return "interacted with your " + entity
	}
}
