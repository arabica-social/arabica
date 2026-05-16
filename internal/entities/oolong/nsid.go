// Package oolong provides typed Go models, atproto record conversions, and
// entity descriptors for the oolong tea-tracking sister app. NSIDs and
// behavior live here (not in the shared internal/atproto or internal/lexicons
// packages) because they are oolong-specific. Arabica has a sibling package
// at internal/entities/arabica with the same shape.
package oolong

const (
	// NSIDBase is the base namespace for all Oolong-owned lexicons.
	NSIDBase = "social.oolong.alpha"

	// SocialNSIDBase is the namespace whose like + comment lexicons
	// oolong reuses. Oolong intentionally does not define its own like
	// or comment record types — it federates against arabica's.
	SocialNSIDBase = "social.arabica.alpha"

	NSIDTea     = NSIDBase + ".tea"
	NSIDBrew    = NSIDBase + ".brew"
	NSIDVessel  = NSIDBase + ".vessel"
	NSIDInfuser = NSIDBase + ".infuser"
	NSIDVendor  = NSIDBase + ".vendor"

	// Cafe and Drink lexicons are defined but deferred for the v1 launch.
	// Their NSID constants are kept so descriptor_bridge and other lookup
	// sites continue to compile even while the entities are not registered.
	NSIDCafe  = NSIDBase + ".cafe"
	NSIDDrink = NSIDBase + ".drink"

	// Social NSIDs federate via arabica's lexicons.
	NSIDLike    = SocialNSIDBase + ".like"
	NSIDComment = SocialNSIDBase + ".comment"
)
