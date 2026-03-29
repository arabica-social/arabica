package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"arabica/internal/atproto"
	"arabica/internal/moderation"

	"github.com/stretchr/testify/assert"
)

func authenticatedRequest(did string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx := atproto.ContextWithAuthDID(req.Context(), did)
	return req.WithContext(ctx)
}

func unauthenticatedRequest() *http.Request {
	return httptest.NewRequest(http.MethodPost, "/", nil)
}

// setupService creates a moderation service from a temp config file with an
// admin (did:plc:admin) and a moderator (did:plc:mod) who can only hide records.
func setupService(t *testing.T) *moderation.Service {
	t.Helper()
	config := `{
		"roles": {
			"admin": {
				"description": "Full access",
				"permissions": ["hide_record", "unhide_record", "blacklist_user", "unblacklist_user", "view_reports", "dismiss_report", "view_audit_log", "reset_autohide"]
			},
			"moderator": {
				"description": "Limited",
				"permissions": ["hide_record", "view_reports"]
			}
		},
		"users": [
			{"did": "did:plc:admin", "handle": "admin", "role": "admin"},
			{"did": "did:plc:mod", "handle": "mod", "role": "moderator"}
		]
	}`
	path := filepath.Join(t.TempDir(), "mod.json")
	err := os.WriteFile(path, []byte(config), 0644)
	assert.NoError(t, err)

	svc, err := moderation.NewService(path)
	assert.NoError(t, err)
	return svc
}

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestRequirePermission(t *testing.T) {
	svc := setupService(t)

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequirePermission(svc, moderation.PermissionHideRecord, okHandler)
		h.ServeHTTP(rec, unauthenticatedRequest())
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("no permission returns 403", func(t *testing.T) {
		rec := httptest.NewRecorder()
		// mod doesn't have blacklist_user
		h := RequirePermission(svc, moderation.PermissionBlacklistUser, okHandler)
		h.ServeHTTP(rec, authenticatedRequest("did:plc:mod"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("unknown user returns 403", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequirePermission(svc, moderation.PermissionHideRecord, okHandler)
		h.ServeHTTP(rec, authenticatedRequest("did:plc:nobody"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("permitted user passes through", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequirePermission(svc, moderation.PermissionHideRecord, okHandler)
		h.ServeHTTP(rec, authenticatedRequest("did:plc:mod"))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("admin has all permissions", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequirePermission(svc, moderation.PermissionBlacklistUser, okHandler)
		h.ServeHTTP(rec, authenticatedRequest("did:plc:admin"))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("nil service returns 403", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequirePermission(nil, moderation.PermissionHideRecord, okHandler)
		h.ServeHTTP(rec, authenticatedRequest("did:plc:admin"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})
}

func TestRequireModerator(t *testing.T) {
	svc := setupService(t)

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequireModerator(svc, okHandler)
		h.ServeHTTP(rec, unauthenticatedRequest())
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("non-moderator returns 403", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequireModerator(svc, okHandler)
		h.ServeHTTP(rec, authenticatedRequest("did:plc:nobody"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("moderator passes through", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequireModerator(svc, okHandler)
		h.ServeHTTP(rec, authenticatedRequest("did:plc:mod"))
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("admin passes through", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequireModerator(svc, okHandler)
		h.ServeHTTP(rec, authenticatedRequest("did:plc:admin"))
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}

func TestRequireAdmin(t *testing.T) {
	svc := setupService(t)

	t.Run("unauthenticated returns 401", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequireAdmin(svc, okHandler)
		h.ServeHTTP(rec, unauthenticatedRequest())
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("moderator returns 403", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequireAdmin(svc, okHandler)
		h.ServeHTTP(rec, authenticatedRequest("did:plc:mod"))
		assert.Equal(t, http.StatusForbidden, rec.Code)
	})

	t.Run("admin passes through", func(t *testing.T) {
		rec := httptest.NewRecorder()
		h := RequireAdmin(svc, okHandler)
		h.ServeHTTP(rec, authenticatedRequest("did:plc:admin"))
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
