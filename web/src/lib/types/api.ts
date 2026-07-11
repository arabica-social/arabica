// Hand-maintained types for profile, notifications, settings, explore,
// and onboarding JSON APIs. See docs/api/ for each endpoint's spec.

import type { Bean, Brew, Brewer, Grinder, Roaster, Recipe } from "./entity_view";

// ===== Profile =====

export type ProfileResponse = {
	profile: { handle: string; display_name: string; avatar: string };
	did: string;
	is_own_profile: boolean;
	is_authenticated: boolean;
	is_app_user: boolean;
	brews: Brew[];
	total_brews: number;
	brews_has_more: boolean;
	brews_next_offset: number;
	beans: Bean[];
	roasters: Roaster[];
	grinders: Grinder[];
	brewers: Brewer[];
	brew_like_counts: Record<string, number>;
	brew_comment_counts: Record<string, number>;
	brew_liked_by_user: Record<string, boolean>;
	brew_cids: Record<string, string>;
	bean_brew_counts: Record<string, number>;
	grinder_brew_counts: Record<string, number>;
	brewer_brew_counts: Record<string, number>;
	roaster_bean_counts: Record<string, number>;
	bean_avg_brew_ratings: Record<string, number>;
	roaster_avg_brew_ratings: Record<string, number>;
};

// ===== Notifications =====

export type NotificationItem = {
	id: string;
	type: string;
	actor_did: string;
	subject_uri: string;
	created_at: string;
	read: boolean;
	actor_handle: string;
	actor_display_name: string;
	actor_avatar: string;
	link: string;
	action_text: string;
};

export type NotificationsResponse = {
	notifications: NotificationItem[];
	next_cursor: string;
};

// ===== Settings =====

export type ProfileStatsVisibility = {
	bean_avg_rating: string;
	roaster_avg_rating: string;
};

export type UserPreferences = {
	temperature_unit: string;
};

export type BlueskyProfileSettings = {
	has_scopes: boolean;
	display_name: string;
	avatar_url: string;
	load_error: string;
	needs_auth_again: boolean;
};

export type SettingsResponse = {
	profile_stats_visibility: ProfileStatsVisibility;
	user_preferences: UserPreferences;
	bluesky_profile: BlueskyProfileSettings;
};

// ===== Explore =====

export type ExploreDocument = {
	URI: string;
	DID: string;
	RecordType: string;
	ClusterKey: string;
	Title: string;
	Summary: string;
	OwnRating: { Valid: boolean; Float64: number };
	SourceRefCount: number;
	CreatedAt: string;
};

export type ExploreFacetCount = {
	Field: string;
	Value: string;
	Count: number;
};

export type ExploreHealth = {
	Ready: boolean;
	Dirty: boolean;
	TotalDocuments: number;
};

export type ExploreResponse = {
	items: import("./feed").FeedItem[];
	documents: Record<string, ExploreDocument>;
	facet_counts: ExploreFacetCount[];
	next_cursor: string;
	health: ExploreHealth;
};

// ===== Admin =====

export type HiddenRecord = {
	at_uri: string;
	hidden_at: string;
	hidden_by: string;
	reason: string;
	auto_hidden: boolean;
};

export type BlacklistedUser = {
	did: string;
	blacklisted_at: string;
	blacklisted_by: string;
	reason: string;
};

export type Report = {
	id: string;
	subject_uri: string;
	subject_did: string;
	reporter_did: string;
	reason: string;
	created_at: string;
	status: string;
	resolved_by?: string;
	resolved_at?: string;
};

export type EnrichedReport = {
	Report: Report;
	OwnerHandle: string;
	ReporterHandle: string;
	PostContent: string;
};

export type AuditEntry = {
	id: string;
	action: string;
	actor_did: string;
	target_uri: string;
	reason: string;
	details?: Record<string, string>;
	timestamp: string;
	auto_mod: boolean;
};

export type Label = {
	id: string;
	entity_type: string;
	entity_id: string;
	label: string;
	value?: string;
	created_at: string;
	created_by: string;
	expires_at?: string;
};

export type AdminStats = {
	KnownUsers: number;
	RegisteredUsers: number;
	IndexedRecords: number;
	TotalLikes: number;
	TotalComments: number;
	FirehoseConnected: boolean;
	RecordsByCollection: Record<string, number>;
};

export type SourceStatus = {
	Source: string;
	LastRun: string;
	LastSuccess: string;
	LastFailure: string;
	LastError: string;
	LastDuration: number; // nanoseconds
	LastSize: number;
	RetainedCount: number;
	NextRun: string;
};

export type AdminResponse = {
	HiddenRecords: HiddenRecord[];
	AuditLog: AuditEntry[];
	Reports: EnrichedReport[];
	BlockedUsers: BlacklistedUser[];
	Labels: Label[];
	Stats: AdminStats;
	Backups: SourceStatus[];
	CanHide: boolean;
	CanUnhide: boolean;
	CanViewLogs: boolean;
	CanViewReports: boolean;
	CanBlock: boolean;
	CanUnblock: boolean;
	CanResetAutoHide: boolean;
	CanManageLabels: boolean;
	IsAdmin: boolean;
};

export type AdminMutationResponse = {
	ok: boolean;
	action: string;
	message?: string;
};

export type ReadinessStatus = {
	HasBean: boolean;
	HasBrewer: boolean;
	HasRoaster: boolean;
};

export type OnboardingResponse = {
	readiness: ReadinessStatus;
	beans: Bean[];
	brewers: Brewer[];
	grinders: Grinder[];
	roasters: Roaster[];
};

export type IncompleteRecord = {
	EntityType: string;
	RKey: string;
	Name: string;
	MissingFields: string[];
};

export type IncompleteRecordsResponse = {
	records: IncompleteRecord[];
};
