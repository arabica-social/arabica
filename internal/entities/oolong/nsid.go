// Package oolong provides typed Go models, atproto record conversions, and
// entity descriptors for the oolong tea-tracking sister app. NSIDs and
// behavior live here (not in the shared internal/atproto or internal/lexicons
// packages) because they are oolong-specific. Arabica has a sibling package
// at internal/entities/arabica with the same shape.
package oolong

const (
	// NSIDBase is the base namespace for all Oolong lexicons.
	NSIDBase = "social.oolong.alpha"

	NSIDTea     = NSIDBase + ".tea"
	NSIDBrew    = NSIDBase + ".brew"
	NSIDBrewer  = NSIDBase + ".brewer"
	NSIDRecipe  = NSIDBase + ".recipe"
	NSIDVendor  = NSIDBase + ".vendor"
	NSIDCafe    = NSIDBase + ".cafe"
	NSIDDrink   = NSIDBase + ".drink"
	NSIDComment = NSIDBase + ".comment"
	NSIDLike    = NSIDBase + ".like"
)
