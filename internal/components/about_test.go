package components

import (
	"arabica/internal/bff"
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestAboutComponent(t *testing.T) {
	// Create test data
	data := &LayoutData{
		Title:           "About",
		IsAuthenticated: false,
		UserDID:         "",
		UserProfile:     nil,
	}

	// Render the component
	var buf bytes.Buffer
	err := About(data).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("Failed to render About component: %v", err)
	}

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
			if !strings.Contains(html, tt.content) {
				t.Errorf("Expected HTML to contain %q, but it was not found", tt.content)
			}
		})
	}
}

func TestAboutComponentAuthenticated(t *testing.T) {
	// Create test data with authenticated user
	data := &LayoutData{
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
	if err != nil {
		t.Fatalf("Failed to render About component: %v", err)
	}

	html := buf.String()

	// When authenticated, should show "Log Your Next Brew" instead of "Get Started"
	if !strings.Contains(html, "Log Your Next Brew") {
		t.Error("Expected authenticated view to show 'Log Your Next Brew' button")
	}

	if strings.Contains(html, "Get Started") {
		t.Error("Expected authenticated view NOT to show 'Get Started' button")
	}
}
