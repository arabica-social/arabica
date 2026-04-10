package middleware

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/moderation"

	"github.com/rs/zerolog/log"
)

// RequirePermission returns middleware that checks the authenticated user has
// the given permission. Returns 401 if unauthenticated, 403 if not permitted.
func RequirePermission(svc *moderation.Service, perm moderation.Permission, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userDID, err := atproto.GetAuthenticatedDID(r.Context())
		if err != nil || userDID == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		if svc == nil || !svc.HasPermission(userDID, perm) {
			log.Warn().
				Str("did", userDID).
				Str("permission", string(perm)).
				Str("path", r.URL.Path).
				Msg("Denied: insufficient permissions")
			http.Error(w, "Permission denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireModerator returns middleware that checks the authenticated user is a
// moderator (any role). Returns 401 if unauthenticated, 403 if not a moderator.
func RequireModerator(svc *moderation.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userDID, err := atproto.GetAuthenticatedDID(r.Context())
		if err != nil || userDID == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		if svc == nil || !svc.IsModerator(userDID) {
			log.Warn().
				Str("did", userDID).
				Str("path", r.URL.Path).
				Msg("Denied: not a moderator")
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequireAdmin returns middleware that checks the authenticated user is an admin.
// Returns 401 if unauthenticated, 403 if not an admin.
func RequireAdmin(svc *moderation.Service, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userDID, err := atproto.GetAuthenticatedDID(r.Context())
		if err != nil || userDID == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}

		if svc == nil || !svc.IsAdmin(userDID) {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
