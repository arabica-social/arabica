package moderation

import "time"

// Permission represents a moderation action that can be performed
type Permission string

const (
	PermissionHideRecord       Permission = "hide_record"
	PermissionUnhideRecord     Permission = "unhide_record"
	PermissionBlacklistUser    Permission = "blacklist_user"
	PermissionUnblacklistUser  Permission = "unblacklist_user"
	PermissionViewReports      Permission = "view_reports"
	PermissionDismissReport    Permission = "dismiss_report"
	PermissionViewAuditLog     Permission = "view_audit_log"
)

// AllPermissions returns all available permissions
func AllPermissions() []Permission {
	return []Permission{
		PermissionHideRecord,
		PermissionUnhideRecord,
		PermissionBlacklistUser,
		PermissionUnblacklistUser,
		PermissionViewReports,
		PermissionDismissReport,
		PermissionViewAuditLog,
	}
}

// RoleName represents the name of a moderation role
type RoleName string

const (
	RoleAdmin     RoleName = "admin"
	RoleModerator RoleName = "moderator"
)

// Role defines a set of permissions for moderators
type Role struct {
	Name        RoleName     `json:"-"` // Set from map key during loading
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
}

// HasPermission checks if this role has the given permission
func (r *Role) HasPermission(perm Permission) bool {
	for _, p := range r.Permissions {
		if p == perm {
			return true
		}
	}
	return false
}

// ModeratorUser represents a user with moderation privileges
type ModeratorUser struct {
	DID    string   `json:"did"`
	Handle string   `json:"handle,omitempty"`
	Role   RoleName `json:"role"`
	Note   string   `json:"note,omitempty"`
}

// Config represents the moderation configuration loaded from JSON
type Config struct {
	Roles map[RoleName]*Role `json:"roles"`
	Users []ModeratorUser    `json:"users"`
}

// Validate checks that the config is valid
func (c *Config) Validate() error {
	if c.Roles == nil {
		c.Roles = make(map[RoleName]*Role)
	}

	// Validate that all users reference valid roles
	for _, user := range c.Users {
		if _, ok := c.Roles[user.Role]; !ok {
			return &ConfigError{
				Field:   "users",
				Message: "user " + user.DID + " references unknown role: " + string(user.Role),
			}
		}
	}

	// Set role names from map keys
	for name, role := range c.Roles {
		role.Name = name
	}

	return nil
}

// ConfigError represents a configuration validation error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return "moderation config error in " + e.Field + ": " + e.Message
}

// HiddenRecord represents a record that has been hidden from the feed
type HiddenRecord struct {
	ATURI      string    `json:"at_uri"`
	HiddenAt   time.Time `json:"hidden_at"`
	HiddenBy   string    `json:"hidden_by"`   // DID of moderator
	Reason     string    `json:"reason"`
	AutoHidden bool      `json:"auto_hidden"` // true if hidden by automod
}

// BlacklistedUser represents a user who has been blacklisted
type BlacklistedUser struct {
	DID           string    `json:"did"`
	BlacklistedAt time.Time `json:"blacklisted_at"`
	BlacklistedBy string    `json:"blacklisted_by"` // DID of admin
	Reason        string    `json:"reason"`
}

// ReportStatus represents the status of a user report
type ReportStatus string

const (
	ReportStatusPending   ReportStatus = "pending"
	ReportStatusDismissed ReportStatus = "dismissed"
	ReportStatusActioned  ReportStatus = "actioned"
)

// Report represents a user report on content
type Report struct {
	ID          string       `json:"id"`          // TID
	SubjectURI  string       `json:"subject_uri"` // AT-URI of reported content
	SubjectDID  string       `json:"subject_did"` // DID of content owner
	ReporterDID string       `json:"reporter_did"`
	Reason      string       `json:"reason"`
	CreatedAt   time.Time    `json:"created_at"`
	Status      ReportStatus `json:"status"`
	ResolvedBy  string       `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time   `json:"resolved_at,omitempty"`
}

// AuditAction represents a type of moderation action
type AuditAction string

const (
	AuditActionHideRecord       AuditAction = "hide_record"
	AuditActionUnhideRecord     AuditAction = "unhide_record"
	AuditActionBlacklistUser    AuditAction = "blacklist_user"
	AuditActionUnblacklistUser  AuditAction = "unblacklist_user"
	AuditActionDismissReport      AuditAction = "dismiss_report"
	AuditActionActionReport       AuditAction = "action_report"
	AuditActionDismissJoinRequest AuditAction = "dismiss_join_request"
	AuditActionCreateInvite       AuditAction = "create_invite"
)

// AuditEntry represents a logged moderation action
type AuditEntry struct {
	ID        string            `json:"id"`
	Action    AuditAction       `json:"action"`
	ActorDID  string            `json:"actor_did"`  // DID of moderator/admin or "automod"
	TargetURI string            `json:"target_uri"` // AT-URI or DID being acted upon
	Reason    string            `json:"reason"`
	Details   map[string]string `json:"details,omitempty"` // Structured metadata (e.g. email, ip, message)
	Timestamp time.Time         `json:"timestamp"`
	AutoMod   bool              `json:"auto_mod"` // true if action was automatic
}
