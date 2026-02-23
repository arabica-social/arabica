package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Static assets
		{"/static/css/output.css", "/static/*"},
		{"/static/js/app.js", "/static/*"},

		// Exact routes (no normalization needed)
		{"/", "/"},
		{"/about", "/about"},
		{"/terms", "/terms"},
		{"/login", "/login"},
		{"/manage", "/manage"},
		{"/brews", "/brews"},

		// Brews with IDs
		{"/brews/abc123", "/brews/:id"},
		{"/brews/abc123/edit", "/brews/:id/edit"},
		{"/brews/new", "/brews/new"},
		{"/brews/export", "/brews/export"},

		// Entity record views
		{"/beans/abc123", "/beans/:id"},
		{"/roasters/abc123", "/roasters/:id"},
		{"/grinders/abc123", "/grinders/:id"},
		{"/brewers/abc123", "/brewers/:id"},

		// Profile
		{"/profile/someone.bsky.social", "/profile/:actor"},

		// API entity routes
		{"/api/beans/abc123", "/api/beans/:id"},
		{"/api/roasters/abc123", "/api/roasters/:id"},
		{"/api/grinders/abc123", "/api/grinders/:id"},
		{"/api/brewers/abc123", "/api/brewers/:id"},
		{"/api/comments/abc123", "/api/comments/:id"},

		// API profile
		{"/api/profile/someone.bsky.social", "/api/profile/:actor"},

		// Modal routes
		{"/api/modals/bean/new", "/api/modals/bean/new"},
		{"/api/modals/bean/abc123", "/api/modals/bean/:id"},
		{"/api/modals/grinder/new", "/api/modals/grinder/new"},
		{"/api/modals/grinder/abc123", "/api/modals/grinder/:id"},

		// Routes that shouldn't be normalized
		{"/api/feed", "/api/feed"},
		{"/api/brews", "/api/brews"},
		{"/api/manage", "/api/manage"},
		{"/api/resolve-handle", "/api/resolve-handle"},
		{"/metrics", "/metrics"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizePath(tt.input))
		})
	}
}
