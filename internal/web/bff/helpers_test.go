package bff

import (
	"testing"
	"time"

	"arabica/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestFormatTemp(t *testing.T) {
	tests := []struct {
		name     string
		temp     float64
		expected string
	}{
		{"zero returns N/A", 0, "N/A"},
		{"celsius range", 93.5, "93.5°C"},
		{"celsius whole number", 90.0, "90.0°C"},
		{"celsius at 100", 100.0, "100.0°C"},
		{"fahrenheit range", 200.0, "200.0°F"},
		{"fahrenheit at 212", 212.0, "212.0°F"},
		{"low temp celsius", 20.5, "20.5°C"},
		{"just over 100 is fahrenheit", 100.1, "100.1°F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTemp(tt.temp)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{"zero returns N/A", 0, "N/A"},
		{"seconds only", 30, "30s"},
		{"exactly one minute", 60, "1m"},
		{"minutes and seconds", 90, "1m 30s"},
		{"multiple minutes", 180, "3m"},
		{"multiple minutes and seconds", 185, "3m 5s"},
		{"large time", 3661, "61m 1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTime(tt.seconds)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestFormatRating(t *testing.T) {
	tests := []struct {
		name     string
		rating   int
		expected string
	}{
		{"zero returns N/A", 0, "N/A"},
		{"rating 1", 1, "1/10"},
		{"rating 5", 5, "5/10"},
		{"rating 10", 10, "10/10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRating(tt.rating)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestPoursToJSON(t *testing.T) {
	tests := []struct {
		name     string
		pours    []*models.Pour
		expected string
	}{
		{
			name:     "empty pours",
			pours:    []*models.Pour{},
			expected: "[]",
		},
		{
			name:     "nil pours",
			pours:    nil,
			expected: "[]",
		},
		{
			name: "single pour",
			pours: []*models.Pour{
				{WaterAmount: 50, TimeSeconds: 30},
			},
			expected: `[{"water":50,"time":30}]`,
		},
		{
			name: "multiple pours",
			pours: []*models.Pour{
				{WaterAmount: 50, TimeSeconds: 30},
				{WaterAmount: 100, TimeSeconds: 60},
				{WaterAmount: 150, TimeSeconds: 90},
			},
			expected: `[{"water":50,"time":30},{"water":100,"time":60},{"water":150,"time":90}]`,
		},
		{
			name: "zero values",
			pours: []*models.Pour{
				{WaterAmount: 0, TimeSeconds: 0},
			},
			expected: `[{"water":0,"time":0}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PoursToJSON(tt.pours)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestHasTemp(t *testing.T) {
	assert.False(t, HasTemp(0))
	assert.False(t, HasTemp(-1))
	assert.True(t, HasTemp(0.1))
	assert.True(t, HasTemp(93.5))
}

func TestHasValue(t *testing.T) {
	assert.False(t, HasValue(0))
	assert.False(t, HasValue(-1))
	assert.True(t, HasValue(1))
	assert.True(t, HasValue(250))
}

func TestSafeAvatarURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"trusted bsky CDN", "https://cdn.bsky.app/img/avatar/did:plc:abc/cid@jpeg", "https://cdn.bsky.app/img/avatar/did:plc:abc/cid@jpeg"},
		{"trusted av-cdn", "https://av-cdn.bsky.app/img/avatar/abc", "https://av-cdn.bsky.app/img/avatar/abc"},
		{"static path", "/static/icon-placeholder.svg", "/static/icon-placeholder.svg"},
		{"non-static relative path", "/evil/path", ""},
		{"http scheme rejected", "http://cdn.bsky.app/img/avatar/abc", ""},
		{"untrusted domain", "https://evil.com/avatar.jpg", ""},
		{"javascript scheme", "javascript:alert(1)", ""},
		{"data URI rejected", "data:image/svg+xml,<svg></svg>", ""},
		{"invalid URL", "://invalid", ""},
		{"subdomain of trusted", "https://sub.cdn.bsky.app/avatar.jpg", "https://sub.cdn.bsky.app/avatar.jpg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, SafeAvatarURL(tt.input))
		})
	}
}

func TestSafeWebsiteURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"valid https", "https://example.com", "https://example.com"},
		{"valid http", "http://example.com", "http://example.com"},
		{"javascript scheme", "javascript:alert(1)", ""},
		{"ftp scheme", "ftp://files.example.com", ""},
		{"no dot in host", "https://localhost", ""},
		{"invalid URL", "://invalid", ""},
		{"https with path", "https://roaster.coffee/about", "https://roaster.coffee/about"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, SafeWebsiteURL(tt.input))
		})
	}
}

func TestEscapeJS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"no special chars", "hello world", "hello world"},
		{"single quotes", "it's a test", "it\\'s a test"},
		{"double quotes", `say "hello"`, `say \"hello\"`},
		{"newlines", "line1\nline2", "line1\\nline2"},
		{"carriage return", "line1\rline2", "line1\\rline2"},
		{"tabs", "col1\tcol2", "col1\\tcol2"},
		{"backslash", `path\to\file`, `path\\to\\file`},
		{"mixed", "it's a \"test\"\nwith\\stuff", "it\\'s a \\\"test\\\"\\nwith\\\\stuff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, EscapeJS(tt.input))
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{"just now", now.Add(-30 * time.Second), "just now"},
		{"1 minute ago", now.Add(-1 * time.Minute), "1 minute ago"},
		{"5 minutes ago", now.Add(-5 * time.Minute), "5 minutes ago"},
		{"1 hour ago", now.Add(-1 * time.Hour), "1 hour ago"},
		{"3 hours ago", now.Add(-3 * time.Hour), "3 hours ago"},
		{"yesterday", now.Add(-36 * time.Hour), "yesterday"},
		{"3 days ago", now.Add(-3 * 24 * time.Hour), "3 days ago"},
		{"1 week ago", now.Add(-8 * 24 * time.Hour), "1 week ago"},
		{"3 weeks ago", now.Add(-22 * 24 * time.Hour), "3 weeks ago"},
		{"1 month ago", now.Add(-35 * 24 * time.Hour), "1 month ago"},
		{"6 months ago", now.Add(-180 * 24 * time.Hour), "6 months ago"},
		{"1 year ago", now.Add(-400 * 24 * time.Hour), "1 year ago"},
		{"2 years ago", now.Add(-800 * 24 * time.Hour), "2 years ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, FormatTimeAgo(tt.input))
		})
	}
}
