package arabica

// NSID (Namespaced Identifier) constants for Arabica lexicons.
// The domain is reversed following ATProto conventions: arabica.social -> social.arabica.
// Using "alpha" namespace during development - will migrate to stable namespace later.
//
// These live with the entity types (not in the generic atproto package) because
// they are arabica-specific; sister apps (oolong, etc.) define their own NSID
// base + collections in their own entity package.
const (
	// NSIDBase is the base namespace for all Arabica lexicons.
	NSIDBase = "social.arabica.alpha"

	// Collection NSIDs.
	NSIDBean    = NSIDBase + ".bean"
	NSIDBrew    = NSIDBase + ".brew"
	NSIDBrewer  = NSIDBase + ".brewer"
	NSIDComment = NSIDBase + ".comment"
	NSIDGrinder = NSIDBase + ".grinder"
	NSIDLike    = NSIDBase + ".like"
	NSIDRecipe  = NSIDBase + ".recipe"
	NSIDRoaster = NSIDBase + ".roaster"
)
