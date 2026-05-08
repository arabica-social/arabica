package atproto

import (
	"regexp"

	"tangled.org/pdewey.com/atp"
)

// MaxRKeyLength is the maximum allowed length for a record key.
const MaxRKeyLength = 512

// rkeyRegex validates AT Protocol record keys (rkeys).
// Valid rkeys contain only alphanumeric characters, hyphens, underscores, colons, and periods.
// They must start with an alphanumeric character and be 1-512 characters long.
// TIDs are the most common format: 13 lowercase base32 characters (e.g., "3kfk4slgu6s2h").
var rkeyRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._:-]{0,511}$`)

// ValidateRKey checks if an rkey is valid according to AT Protocol spec.
// Returns true if valid, false otherwise.
func ValidateRKey(rkey string) bool {
	if rkey == "" || len(rkey) > MaxRKeyLength {
		return false
	}
	// Reserved rkeys that should not be used
	if rkey == "." || rkey == ".." {
		return false
	}
	return rkeyRegex.MatchString(rkey)
}

// BuildATURI constructs an AT-URI from a DID, collection NSID, and record key.
var BuildATURI = atp.BuildATURI

// ExtractRKeyFromURI extracts the record key from an AT-URI.
// Returns the rkey if successful, empty string if parsing fails.
var ExtractRKeyFromURI = atp.RKeyFromURI
