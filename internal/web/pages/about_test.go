package pages

import (
	"arabica/internal/web/bff"
	"arabica/internal/web/components"
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAboutComponent(t *testing.T) {
	// Create test data
	data := &components.LayoutData{
		Title:           "About",
		IsAuthenticated: false,
		UserDID:         "",
		UserProfile:     nil,
	}

	// Render the component
	var buf bytes.Buffer
	err := About(data).Render(context.Background(), &buf)
	assert.NoError(t, err)

	html := buf.String()

	// Verify key content is present
	tests := []struct {
		name    string
		content string
	}{
		{"title", "About Arabica"},
		{"alpha badge", "ALPHA"},
		{"main heading", "Your Coffee Journey, Your Data"},
		{"AT Protocol mention", "AT Protocol"},
		{"footer", "Your brew, your data"},
		{"get started button", "Get Started"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, html, tt.content)
		})
	}
}

func TestAboutComponentAuthenticated(t *testing.T) {
	// Create test data with authenticated user
	data := &components.LayoutData{
		Title:           "About",
		IsAuthenticated: true,
		UserDID:         "did:plc:test123",
		UserProfile: &bff.UserProfile{
			Handle:      "testuser.bsky.social",
			DisplayName: "Test User",
			Avatar:      "",
		},
	}

	// Render the component
	var buf bytes.Buffer
	err := About(data).Render(context.Background(), &buf)
	assert.NoError(t, err)

	html := buf.String()

	// When authenticated, should show "Log Your Next Brew" instead of "Get Started"
	assert.Contains(t, html, "Back to Home")
	assert.NotContains(t, html, "Get Started")
}
