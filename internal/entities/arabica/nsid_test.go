// External test package to avoid an internal-test cycle: atproto/cache.go
// imports arabica (for record types), and the BuildATURI test wants to call
// atproto.BuildATURI on arabica's NSID constants.
package arabica_test

import (
	"testing"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
)

func TestNSIDConstants(t *testing.T) {
	if arabica.NSIDBase != "social.arabica.alpha" {
		t.Errorf("NSIDBase = %q, want %q", arabica.NSIDBase, "social.arabica.alpha")
	}

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"NSIDBean", arabica.NSIDBean, "social.arabica.alpha.bean"},
		{"NSIDBrew", arabica.NSIDBrew, "social.arabica.alpha.brew"},
		{"NSIDBrewer", arabica.NSIDBrewer, "social.arabica.alpha.brewer"},
		{"NSIDGrinder", arabica.NSIDGrinder, "social.arabica.alpha.grinder"},
		{"NSIDRoaster", arabica.NSIDRoaster, "social.arabica.alpha.roaster"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestBuildATURI_WithNSIDConstants(t *testing.T) {
	did := "did:plc:testuser"
	rkey := "abc123"

	tests := []struct {
		collection string
		expected   string
	}{
		{arabica.NSIDBean, "at://did:plc:testuser/social.arabica.alpha.bean/abc123"},
		{arabica.NSIDBrew, "at://did:plc:testuser/social.arabica.alpha.brew/abc123"},
		{arabica.NSIDBrewer, "at://did:plc:testuser/social.arabica.alpha.brewer/abc123"},
		{arabica.NSIDGrinder, "at://did:plc:testuser/social.arabica.alpha.grinder/abc123"},
		{arabica.NSIDRoaster, "at://did:plc:testuser/social.arabica.alpha.roaster/abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.collection, func(t *testing.T) {
			got := atproto.BuildATURI(did, tt.collection, rkey)
			if got != tt.expected {
				t.Errorf("BuildATURI with %s = %q, want %q", tt.collection, got, tt.expected)
			}
		})
	}
}
