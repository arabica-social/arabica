package moderation

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
)

// Service provides moderation functionality with role-based access control
type Service struct {
	mu         sync.RWMutex
	config     *Config
	configPath string

	// Quick lookup maps built from config
	userRoles map[string]*Role // DID -> Role
	userInfos map[string]*ModeratorUser // DID -> ModeratorUser
}

// NewService creates a new moderation service.
// If configPath is empty, the service will be in "disabled" mode
// where all permission checks return false.
func NewService(configPath string) (*Service, error) {
	s := &Service{
		configPath: configPath,
		userRoles:  make(map[string]*Role),
		userInfos:  make(map[string]*ModeratorUser),
	}

	if configPath == "" {
		log.Info().Msg("moderation: no config path provided, service disabled")
		return s, nil
	}

	if err := s.loadConfig(); err != nil {
		return nil, fmt.Errorf("failed to load moderation config: %w", err)
	}

	return s, nil
}

// loadConfig reads and parses the config file
func (s *Service) loadConfig() error {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn().Str("path", s.configPath).Msg("moderation: config file not found, service disabled")
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = &config
	s.rebuildLookupMaps()

	log.Info().
		Int("roles", len(config.Roles)).
		Int("users", len(config.Users)).
		Str("path", s.configPath).
		Msg("moderation: config loaded")

	return nil
}

// rebuildLookupMaps rebuilds the quick lookup maps from config
// Caller must hold the write lock
func (s *Service) rebuildLookupMaps() {
	s.userRoles = make(map[string]*Role)
	s.userInfos = make(map[string]*ModeratorUser)

	if s.config == nil {
		return
	}

	for i := range s.config.Users {
		user := &s.config.Users[i]
		role, ok := s.config.Roles[user.Role]
		if ok {
			s.userRoles[user.DID] = role
			s.userInfos[user.DID] = user
		}
	}
}

// Reload reloads the configuration from disk
func (s *Service) Reload() error {
	if s.configPath == "" {
		return nil
	}
	return s.loadConfig()
}

// IsEnabled returns true if the moderation service is configured and enabled
func (s *Service) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config != nil && len(s.config.Users) > 0
}

// IsAdmin returns true if the given DID has the admin role
func (s *Service) IsAdmin(did string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	role, ok := s.userRoles[did]
	if !ok {
		return false
	}
	return role.Name == RoleAdmin
}

// IsModerator returns true if the given DID has moderator privileges
// This includes both moderators and admins (admins have all moderator permissions)
func (s *Service) IsModerator(did string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.userRoles[did]
	return ok
}

// HasPermission returns true if the given DID has the specified permission
func (s *Service) HasPermission(did string, permission Permission) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	role, ok := s.userRoles[did]
	if !ok {
		return false
	}
	return role.HasPermission(permission)
}

// GetRole returns the role for the given DID, if any
func (s *Service) GetRole(did string) (*Role, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	role, ok := s.userRoles[did]
	if !ok {
		return nil, false
	}
	// Return a copy to prevent external modification
	roleCopy := *role
	return &roleCopy, true
}

// GetModeratorUser returns the moderator user info for the given DID, if any
func (s *Service) GetModeratorUser(did string) (*ModeratorUser, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.userInfos[did]
	if !ok {
		return nil, false
	}
	// Return a copy to prevent external modification
	userCopy := *user
	return &userCopy, true
}

// ListModerators returns all configured moderator users
func (s *Service) ListModerators() []ModeratorUser {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]ModeratorUser, len(s.config.Users))
	copy(result, s.config.Users)
	return result
}

// ListRoles returns all configured roles
func (s *Service) ListRoles() map[RoleName]*Role {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config == nil {
		return nil
	}

	// Return a copy to prevent external modification
	result := make(map[RoleName]*Role)
	for name, role := range s.config.Roles {
		roleCopy := *role
		result[name] = &roleCopy
	}
	return result
}

// GetPermissionsForDID returns all permissions for the given DID
func (s *Service) GetPermissionsForDID(did string) []Permission {
	s.mu.RLock()
	defer s.mu.RUnlock()

	role, ok := s.userRoles[did]
	if !ok {
		return nil
	}

	// Return a copy to prevent external modification
	result := make([]Permission, len(role.Permissions))
	copy(result, role.Permissions)
	return result
}
