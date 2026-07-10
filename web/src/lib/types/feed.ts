// Hand-maintained types for the feed JSON API.
// Mirrors internal/handlers/feed.go (FeedJSONResponse) and
// internal/feed/service.go (FeedItem). See docs/api/feed.md.

import type {
	Bean,
	Brew,
	Brewer,
	Grinder,
	Recipe,
	Roaster,
	AuthorSummary,
} from "./entity_view";

export type RecordType =
	| "brew"
	| "bean"
	| "roaster"
	| "grinder"
	| "brewer"
	| "recipe";

export type FeedAuthor = AuthorSummary;

export type FeedItem = {
	record_type: RecordType;
	action: string;
	// The typed record. Narrow with the record_type discriminator.
	record: Record<string, unknown>;
	author: FeedAuthor;
	timestamp: string;
	time_ago: string;
	like_count: number;
	comment_count: number;
	subject_uri: string;
	subject_cid: string;
	is_liked_by_viewer: boolean;
	is_owner: boolean;
	// Moderation context for moderator tooling. Only populated for
	// moderator viewers; falsy for everyone else.
	is_moderator: boolean;
	can_hide_record: boolean;
	can_block_user: boolean;
	is_record_hidden: boolean;
	author_did: string;
};

export type FeedResponse = {
	items: FeedItem[];
	next_cursor: string;
	is_authenticated: boolean;
	query: {
		type: string;
		sort: string;
	};
	tabs: FeedFilterTab[];
};

export type FeedFilterTab = {
	label: string;
	value: string;
};

// Type guards to narrow the `record` union field by record_type.
export function isBrewItem(item: FeedItem): item is FeedItem & { record: Brew } {
	return item.record_type === "brew";
}
export function isBeanItem(item: FeedItem): item is FeedItem & { record: Bean } {
	return item.record_type === "bean";
}
export function isRoasterItem(
	item: FeedItem,
): item is FeedItem & { record: Roaster } {
	return item.record_type === "roaster";
}
export function isGrinderItem(
	item: FeedItem,
): item is FeedItem & { record: Grinder } {
	return item.record_type === "grinder";
}
export function isBrewerItem(
	item: FeedItem,
): item is FeedItem & { record: Brewer } {
	return item.record_type === "brewer";
}
export function isRecipeItem(
	item: FeedItem,
): item is FeedItem & { record: Recipe } {
	return item.record_type === "recipe";
}
