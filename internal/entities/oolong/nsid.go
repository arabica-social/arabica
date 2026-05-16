// Package oolong provides typed Go models, atproto record conversions, and
// entity descriptors for the oolong tea-tracking sister app. NSIDs and
// behavior live here (not in the shared internal/atproto or internal/lexicons
// packages) because they are oolong-specific. Arabica has a sibling package
// at internal/entities/arabica with the same shape.
package oolong

const (
	// NSIDBase is the base namespace for all Oolong-owned lexicons,
	// including its own like + comment social lexicons.
	NSIDBase = "social.oolong.alpha"

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

	// Social NSIDs in oolong's own namespace. Mirror arabica's shape
	// minus the lexicon id.
	NSIDLike    = NSIDBase + ".like"
	NSIDComment = NSIDBase + ".comment"
)
