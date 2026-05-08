package atproto

import (
	"strings"
	"testing"
)

func TestValidateRKey(t *testing.T) {
	tests := []struct {
		name  string
		rkey  string
		valid bool
	}{
		// Valid rkeys
		{"TID format", "3kfk4slgu6s2h", true},
		{"short alphanumeric", "abc123", true},
		{"single char", "a", true},
		{"with hyphen", "my-record", true},
		{"with underscore", "my_record", true},
		{"with period", "my.record", true},
		{"with colon", "my:record", true},
		{"mixed valid chars", "a1-b2_c3.d4:e5", true},
		{"uppercase", "ABC123", true},
		{"mixed case", "AbC123xYz", true},

		// Invalid rkeys
		{"empty string", "", false},
		{"starts with hyphen", "-abc", false},
		{"starts with underscore", "_abc", false},
		{"starts with period", ".abc", false},
		{"starts with colon", ":abc", false},
		{"reserved dot", ".", false},
		{"reserved dotdot", "..", false},
		{"contains slash", "abc/def", false},
		{"contains space", "abc def", false},
		{"contains at", "abc@def", false},
		{"contains hash", "abc#def", false},
		{"contains question", "abc?def", false},
		{"too long", strings.Repeat("a", 513), false},
		{"max length valid", strings.Repeat("a", 512), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateRKey(tt.rkey)
			if got != tt.valid {
				t.Errorf("ValidateRKey(%q) = %v, want %v", tt.rkey, got, tt.valid)
			}
		})
	}
}

func TestBuildATURI(t *testing.T) {
	tests := []struct {
		name       string
		did        string
		collection string
		rkey       string
		expected   string
	}{
		{
			name:       "basic URI",
			did:        "did:plc:abc123",
			collection: "social.arabica.alpha.bean",
			rkey:       "3jxyabc",
			expected:   "at://did:plc:abc123/social.arabica.alpha.bean/3jxyabc",
		},
		{
			name:       "web DID",
			did:        "did:web:example.com",
			collection: "social.arabica.alpha.brew",
			rkey:       "xyz789",
			expected:   "at://did:web:example.com/social.arabica.alpha.brew/xyz789",
		},
		{
			name:       "with grinder collection",
			did:        "did:plc:test456",
			collection: "social.arabica.alpha.grinder",
			rkey:       "rkey123",
			expected:   "at://did:plc:test456/social.arabica.alpha.grinder/rkey123",
		},
		{
			name:       "empty rkey",
			did:        "did:plc:abc",
			collection: "social.arabica.alpha.bean",
			rkey:       "",
			expected:   "at://did:plc:abc/social.arabica.alpha.bean/",
		},
		{
			name:       "long TID rkey",
			did:        "did:plc:abc123xyz789",
			collection: "social.arabica.alpha.brew",
			rkey:       "3kfk4slgu6s2h",
			expected:   "at://did:plc:abc123xyz789/social.arabica.alpha.brew/3kfk4slgu6s2h",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildATURI(tt.did, tt.collection, tt.rkey)
			if got != tt.expected {
				t.Errorf("BuildATURI(%q, %q, %q) = %q, want %q",
					tt.did, tt.collection, tt.rkey, got, tt.expected)
			}
		})
	}
}
