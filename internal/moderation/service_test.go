package moderation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService_NoConfig(t *testing.T) {
	// Service should work in disabled mode with empty config path
	svc, err := NewService("")
	require.NoError(t, err)
	assert.NotNil(t, svc)
	assert.False(t, svc.IsEnabled())
	assert.False(t, svc.IsAdmin("did:plc:test"))
	assert.False(t, svc.IsModerator("did:plc:test"))
	assert.False(t, svc.HasPermission("did:plc:test", PermissionHideRecord))
}

func TestNewService_MissingFile(t *testing.T) {
	// Service should work in disabled mode when file doesn't exist
	svc, err := NewService("/nonexistent/path/config.json")
	require.NoError(t, err)
	assert.NotNil(t, svc)
	assert.False(t, svc.IsEnabled())
}

func TestNewService_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "moderators.json")

	err := os.WriteFile(configPath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	_, err = NewService(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestNewService_InvalidRole(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "moderators.json")

	config := `{
		"roles": {
			"admin": {
				"description": "Admin role",
				"permissions": ["hide_record"]
			}
		},
		"users": [
			{"did": "did:plc:test", "role": "nonexistent"}
		]
	}`

	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	_, err = NewService(configPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown role")
}

func TestNewService_ValidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "moderators.json")

	config := `{
		"roles": {
			"admin": {
				"description": "Full platform control",
				"permissions": ["hide_record", "unhide_record", "blacklist_user", "unblacklist_user", "view_reports", "dismiss_report", "view_audit_log"]
			},
			"moderator": {
				"description": "Content moderation",
				"permissions": ["hide_record", "unhide_record", "view_reports", "dismiss_report"]
			}
		},
		"users": [
			{"did": "did:plc:admin1", "handle": "admin.test", "role": "admin", "note": "Test admin"},
			{"did": "did:plc:mod1", "handle": "mod.test", "role": "moderator"}
		]
	}`

	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	svc, err := NewService(configPath)
	require.NoError(t, err)
	assert.True(t, svc.IsEnabled())
}

func createTestService(t *testing.T) *Service {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "moderators.json")

	config := `{
		"roles": {
			"admin": {
				"description": "Full platform control",
				"permissions": ["hide_record", "unhide_record", "blacklist_user", "unblacklist_user", "view_reports", "dismiss_report", "view_audit_log"]
			},
			"moderator": {
				"description": "Content moderation",
				"permissions": ["hide_record", "unhide_record", "view_reports", "dismiss_report"]
			}
		},
		"users": [
			{"did": "did:plc:admin1", "handle": "admin.test", "role": "admin", "note": "Test admin"},
			{"did": "did:plc:mod1", "handle": "mod.test", "role": "moderator"}
		]
	}`

	err := os.WriteFile(configPath, []byte(config), 0644)
	require.NoError(t, err)

	svc, err := NewService(configPath)
	require.NoError(t, err)
	return svc
}

func TestIsAdmin(t *testing.T) {
	svc := createTestService(t)

	assert.True(t, svc.IsAdmin("did:plc:admin1"))
	assert.False(t, svc.IsAdmin("did:plc:mod1"))
	assert.False(t, svc.IsAdmin("did:plc:unknown"))
}

func TestIsModerator(t *testing.T) {
	svc := createTestService(t)

	// Both admins and moderators should return true
	assert.True(t, svc.IsModerator("did:plc:admin1"))
	assert.True(t, svc.IsModerator("did:plc:mod1"))
	assert.False(t, svc.IsModerator("did:plc:unknown"))
}

func TestHasPermission(t *testing.T) {
	svc := createTestService(t)

	// Admin has all permissions
	assert.True(t, svc.HasPermission("did:plc:admin1", PermissionHideRecord))
	assert.True(t, svc.HasPermission("did:plc:admin1", PermissionBlacklistUser))
	assert.True(t, svc.HasPermission("did:plc:admin1", PermissionViewAuditLog))

	// Moderator has limited permissions
	assert.True(t, svc.HasPermission("did:plc:mod1", PermissionHideRecord))
	assert.True(t, svc.HasPermission("did:plc:mod1", PermissionViewReports))
	assert.False(t, svc.HasPermission("did:plc:mod1", PermissionBlacklistUser))
	assert.False(t, svc.HasPermission("did:plc:mod1", PermissionViewAuditLog))

	// Unknown user has no permissions
	assert.False(t, svc.HasPermission("did:plc:unknown", PermissionHideRecord))
}

func TestGetRole(t *testing.T) {
	svc := createTestService(t)

	role, ok := svc.GetRole("did:plc:admin1")
	assert.True(t, ok)
	assert.Equal(t, RoleAdmin, role.Name)
	assert.Equal(t, "Full platform control", role.Description)

	role, ok = svc.GetRole("did:plc:mod1")
	assert.True(t, ok)
	assert.Equal(t, RoleModerator, role.Name)

	_, ok = svc.GetRole("did:plc:unknown")
	assert.False(t, ok)
}

func TestGetModeratorUser(t *testing.T) {
	svc := createTestService(t)

	user, ok := svc.GetModeratorUser("did:plc:admin1")
	assert.True(t, ok)
	assert.Equal(t, "did:plc:admin1", user.DID)
	assert.Equal(t, "admin.test", user.Handle)
	assert.Equal(t, RoleAdmin, user.Role)
	assert.Equal(t, "Test admin", user.Note)

	user, ok = svc.GetModeratorUser("did:plc:mod1")
	assert.True(t, ok)
	assert.Equal(t, "mod.test", user.Handle)
	assert.Empty(t, user.Note)

	_, ok = svc.GetModeratorUser("did:plc:unknown")
	assert.False(t, ok)
}

func TestListModerators(t *testing.T) {
	svc := createTestService(t)

	users := svc.ListModerators()
	assert.Len(t, users, 2)

	// Verify it returns copies (mutation shouldn't affect service)
	users[0].Handle = "mutated"
	originalUser, _ := svc.GetModeratorUser("did:plc:admin1")
	assert.Equal(t, "admin.test", originalUser.Handle)
}

func TestListRoles(t *testing.T) {
	svc := createTestService(t)

	roles := svc.ListRoles()
	assert.Len(t, roles, 2)
	assert.Contains(t, roles, RoleAdmin)
	assert.Contains(t, roles, RoleModerator)
}

func TestGetPermissionsForDID(t *testing.T) {
	svc := createTestService(t)

	perms := svc.GetPermissionsForDID("did:plc:admin1")
	assert.Len(t, perms, 7)
	assert.Contains(t, perms, PermissionHideRecord)
	assert.Contains(t, perms, PermissionBlacklistUser)

	perms = svc.GetPermissionsForDID("did:plc:mod1")
	assert.Len(t, perms, 4)
	assert.Contains(t, perms, PermissionHideRecord)
	assert.NotContains(t, perms, PermissionBlacklistUser)

	perms = svc.GetPermissionsForDID("did:plc:unknown")
	assert.Nil(t, perms)
}

func TestReload(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "moderators.json")

	// Start with one admin
	config1 := `{
		"roles": {
			"admin": {
				"description": "Admin",
				"permissions": ["hide_record"]
			}
		},
		"users": [
			{"did": "did:plc:admin1", "role": "admin"}
		]
	}`
	err := os.WriteFile(configPath, []byte(config1), 0644)
	require.NoError(t, err)

	svc, err := NewService(configPath)
	require.NoError(t, err)

	assert.True(t, svc.IsAdmin("did:plc:admin1"))
	assert.False(t, svc.IsAdmin("did:plc:admin2"))

	// Update config with another admin
	config2 := `{
		"roles": {
			"admin": {
				"description": "Admin",
				"permissions": ["hide_record"]
			}
		},
		"users": [
			{"did": "did:plc:admin1", "role": "admin"},
			{"did": "did:plc:admin2", "role": "admin"}
		]
	}`
	err = os.WriteFile(configPath, []byte(config2), 0644)
	require.NoError(t, err)

	err = svc.Reload()
	require.NoError(t, err)

	assert.True(t, svc.IsAdmin("did:plc:admin1"))
	assert.True(t, svc.IsAdmin("did:plc:admin2"))
}

func TestRole_HasPermission(t *testing.T) {
	role := &Role{
		Name:        RoleModerator,
		Permissions: []Permission{PermissionHideRecord, PermissionViewReports},
	}

	assert.True(t, role.HasPermission(PermissionHideRecord))
	assert.True(t, role.HasPermission(PermissionViewReports))
	assert.False(t, role.HasPermission(PermissionBlacklistUser))
}

func TestConfig_Validate(t *testing.T) {
	t.Run("nil roles map", func(t *testing.T) {
		config := &Config{
			Roles: nil,
			Users: []ModeratorUser{},
		}
		err := config.Validate()
		assert.NoError(t, err)
		assert.NotNil(t, config.Roles)
	})

	t.Run("user with unknown role", func(t *testing.T) {
		config := &Config{
			Roles: map[RoleName]*Role{
				RoleAdmin: {Description: "Admin"},
			},
			Users: []ModeratorUser{
				{DID: "did:plc:test", Role: "unknown"},
			},
		}
		err := config.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown role")
	})

	t.Run("valid config sets role names", func(t *testing.T) {
		config := &Config{
			Roles: map[RoleName]*Role{
				RoleAdmin: {Description: "Admin"},
			},
			Users: []ModeratorUser{
				{DID: "did:plc:test", Role: RoleAdmin},
			},
		}
		err := config.Validate()
		assert.NoError(t, err)
		assert.Equal(t, RoleAdmin, config.Roles[RoleAdmin].Name)
	})
}

func TestDisabledService(t *testing.T) {
	// All methods should be safe to call on a disabled service
	svc, err := NewService("")
	require.NoError(t, err)

	assert.False(t, svc.IsEnabled())
	assert.False(t, svc.IsAdmin("did:plc:any"))
	assert.False(t, svc.IsModerator("did:plc:any"))
	assert.False(t, svc.HasPermission("did:plc:any", PermissionHideRecord))

	role, ok := svc.GetRole("did:plc:any")
	assert.Nil(t, role)
	assert.False(t, ok)

	user, ok := svc.GetModeratorUser("did:plc:any")
	assert.Nil(t, user)
	assert.False(t, ok)

	assert.Nil(t, svc.ListModerators())
	assert.Nil(t, svc.ListRoles())
	assert.Nil(t, svc.GetPermissionsForDID("did:plc:any"))

	// Reload should be a no-op
	assert.NoError(t, svc.Reload())
}
