package atproto

import "tangled.org/pdewey.com/atp"

// BuildATURI constructs an AT-URI from a DID, collection NSID, and record key.
var BuildATURI = atp.BuildATURI

// ExtractRKeyFromURI extracts the record key from an AT-URI.
var ExtractRKeyFromURI = atp.RKeyFromURI
