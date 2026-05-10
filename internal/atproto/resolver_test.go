package atproto

import (
	"testing"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"

	"tangled.org/pdewey.com/atp"
)

func TestParseATURI(t *testing.T) {
	tests := []struct {
		name           string
		uri            string
		wantDID        string
		wantCollection string
		wantRKey       string
		wantErr        bool
	}{
		{
			name:           "valid plc DID URI",
			uri:            "at://did:plc:abc123/social.arabica.alpha.bean/3jxyabc",
			wantDID:        "did:plc:abc123",
			wantCollection: "social.arabica.alpha.bean",
			wantRKey:       "3jxyabc",
			wantErr:        false,
		},
		{
			name:           "valid web DID URI",
			uri:            "at://did:web:example.com/social.arabica.alpha.brew/xyz789",
			wantDID:        "did:web:example.com",
			wantCollection: "social.arabica.alpha.brew",
			wantRKey:       "xyz789",
			wantErr:        false,
		},
		{
			name:           "long TID rkey",
			uri:            "at://did:plc:longtestdid123/social.arabica.alpha.grinder/3kfk4slgu6s2h",
			wantDID:        "did:plc:longtestdid123",
			wantCollection: "social.arabica.alpha.grinder",
			wantRKey:       "3kfk4slgu6s2h",
			wantErr:        false,
		},
		{
			name:           "bsky app collection",
			uri:            "at://did:plc:user123/app.bsky.feed.post/abc123",
			wantDID:        "did:plc:user123",
			wantCollection: "app.bsky.feed.post",
			wantRKey:       "abc123",
			wantErr:        false,
		},
		{
			name:    "invalid scheme",
			uri:     "http://did:plc:abc123/social.arabica.alpha.bean/3jxyabc",
			wantErr: true,
		},
		{
			name:    "missing scheme",
			uri:     "did:plc:abc123/social.arabica.alpha.bean/3jxyabc",
			wantErr: true,
		},
		{
			name:    "empty URI",
			uri:     "",
			wantErr: true,
		},
		{
			name:           "URI without collection/rkey (valid DID reference)",
			uri:            "at://did:plc:abc123",
			wantDID:        "did:plc:abc123",
			wantCollection: "",
			wantRKey:       "",
			wantErr:        false,
		},
		{
			name:    "garbage input",
			uri:     "not a valid uri at all",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := atp.ParseATURI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseATURI(%q) expected error, got nil", tt.uri)
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseATURI(%q) unexpected error = %v", tt.uri, err)
			}

			if got.DID != tt.wantDID {
				t.Errorf("DID = %q, want %q", got.DID, tt.wantDID)
			}
			if got.Collection != tt.wantCollection {
				t.Errorf("Collection = %q, want %q", got.Collection, tt.wantCollection)
			}
			if got.RKey != tt.wantRKey {
				t.Errorf("RKey = %q, want %q", got.RKey, tt.wantRKey)
			}
		})
	}
}

func TestATURIRoundTrip(t *testing.T) {
	tests := []struct {
		did        string
		collection string
		rkey       string
	}{
		{"did:plc:abc123", arabica.NSIDBean, "bean123"},
		{"did:plc:xyz789", arabica.NSIDBrew, "3kfk4slgu6s2h"},
		{"did:web:example.com", arabica.NSIDRoaster, "roaster456"},
		{"did:plc:longdidvalue123456789", arabica.NSIDGrinder, "g1"},
	}

	for _, tt := range tests {
		t.Run(tt.did+"/"+tt.collection+"/"+tt.rkey, func(t *testing.T) {
			uri := atp.BuildATURI(tt.did, tt.collection, tt.rkey)
			parsed, err := atp.ParseATURI(uri)
			if err != nil {
				t.Fatalf("ParseATURI() error = %v", err)
			}
			if parsed.DID != tt.did {
				t.Errorf("DID round-trip: got %q, want %q", parsed.DID, tt.did)
			}
			if parsed.Collection != tt.collection {
				t.Errorf("Collection round-trip: got %q, want %q", parsed.Collection, tt.collection)
			}
			if parsed.RKey != tt.rkey {
				t.Errorf("RKey round-trip: got %q, want %q", parsed.RKey, tt.rkey)
			}
		})
	}
}
