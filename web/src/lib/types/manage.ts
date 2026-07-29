// Hand-maintained types for the manage and brew list JSON APIs.
// Must match internal/arabica/handlers/manage_json.go (ManageResponseJSON,
// ManageStatsJSON) and internal/arabica/handlers/brew.go
// (BrewListJSONResponse). See docs/api/manage.md and docs/api/brews.md.

import type { Bean, Brewer, Grinder, Recipe, Roaster } from "./entity_view";

export type ManageStatsJSON = {
	bean_brew_counts: Record<string, number>;
	grinder_brew_counts: Record<string, number>;
	brewer_brew_counts: Record<string, number>;
	roaster_bean_counts: Record<string, number>;
	bean_avg_brew_ratings: Record<string, number>;
	roaster_avg_brew_ratings: Record<string, number>;
};

export type ManageResponseJSON = {
	did: string;
	beans: Bean[];
	roasters: Roaster[];
	grinders: Grinder[];
	brewers: Brewer[];
	recipes: Recipe[];
	stats: ManageStatsJSON;
};

export type BrewListResponse = {
	brews: import("./entity_view").Brew[];
	has_more: boolean;
	next_offset: number;
};
