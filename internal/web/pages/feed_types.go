package pages

// FeedModerationContext carries the viewer's moderation permissions and the
// set of currently-hidden record URIs. It is populated by
// Handler.BuildModerationContext and consumed by ApplyModerationContext so
// the JSON feed path can surface hide/block controls and hidden-record
// badges for moderator viewers. Non-moderator viewers get zero values.
type FeedModerationContext struct {
	IsModerator   bool            // User has moderator role
	CanHideRecord bool            // User has hide_record permission
	CanBlockUser  bool            // User has blacklist_user permission
	HiddenURIs    map[string]bool // URIs that are currently hidden
}
