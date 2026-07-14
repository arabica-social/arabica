import { deleteJSON, postJSON, putJSON } from "./client";
import type { Bean, Brewer, Brew, Grinder, Recipe, Roaster } from "$lib/types/entity_view";

export type RoasterInput = {
	name: string;
	location: string;
	website: string;
	source_ref?: string;
};

export function createRoaster(fetchFn: typeof fetch, input: RoasterInput): Promise<Roaster> {
	return postJSON<Roaster>(fetchFn, "/api/roasters", input);
}

export function updateRoaster(fetchFn: typeof fetch, rkey: string, input: RoasterInput): Promise<Roaster> {
	return putJSON<Roaster>(fetchFn, `/api/roasters/${encodeURIComponent(rkey)}`, input);
}

export type BeanInput = {
	name: string;
	origin: string;
	variety: string;
	roast_level: string;
	roast_date?: string;
	process: string;
	description: string;
	notes: string;
	link: string;
	roaster_rkey: string;
	rating?: number;
	closed: boolean;
	source_ref?: string;
};

export function createBean(fetchFn: typeof fetch, input: BeanInput): Promise<Bean> {
	return postJSON<Bean>(fetchFn, "/api/beans", input);
}

export function updateBean(fetchFn: typeof fetch, rkey: string, input: BeanInput): Promise<Bean> {
	return putJSON<Bean>(fetchFn, `/api/beans/${encodeURIComponent(rkey)}`, input);
}

export type GrinderInput = {
	name: string;
	grinder_type: string;
	burr_type: string;
	notes: string;
	link: string;
	source_ref?: string;
};

export function createGrinder(fetchFn: typeof fetch, input: GrinderInput): Promise<Grinder> {
	return postJSON<Grinder>(fetchFn, "/api/grinders", input);
}

export function updateGrinder(fetchFn: typeof fetch, rkey: string, input: GrinderInput): Promise<Grinder> {
	return putJSON<Grinder>(fetchFn, `/api/grinders/${encodeURIComponent(rkey)}`, input);
}

export type BrewerInput = {
	name: string;
	brewer_type: string;
	description: string;
	link: string;
	source_ref?: string;
};

export function createBrewer(fetchFn: typeof fetch, input: BrewerInput): Promise<Brewer> {
	return postJSON<Brewer>(fetchFn, "/api/brewers", input);
}

export function updateBrewer(fetchFn: typeof fetch, rkey: string, input: BrewerInput): Promise<Brewer> {
	return putJSON<Brewer>(fetchFn, `/api/brewers/${encodeURIComponent(rkey)}`, input);
}

export type RecipePourInput = {
	water_amount: number;
	time_seconds: number;
};

export type RecipeInput = {
	name: string;
	brewer_rkey: string;
	brewer_type: string;
	coffee_amount: number;
	water_amount: number;
	notes: string;
	pours: RecipePourInput[];
	source_ref?: string;
};

export function createRecipe(fetchFn: typeof fetch, input: RecipeInput): Promise<Recipe> {
	return postJSON<Recipe>(fetchFn, "/api/recipes", input);
}

export function updateRecipe(fetchFn: typeof fetch, rkey: string, input: RecipeInput): Promise<Recipe> {
	return putJSON<Recipe>(fetchFn, `/api/recipes/${encodeURIComponent(rkey)}`, input);
}

export type BrewPourInput = {
	water_amount: number;
	time_seconds: number;
};

export type BrewEspressoParams = {
	yield_weight: number;
	pressure: number;
	pre_infusion_seconds: number;
};

export type BrewPouroverParams = {
	bloom_water: number;
	bloom_seconds: number;
	drawdown_seconds: number;
	bypass_water: number;
	filter: string;
};

export type BrewInput = {
	bean_rkey: string;
	recipe_rkey?: string;
	recipe_owner_did?: string;
	method?: string;
	temperature?: number;
	water_amount?: number;
	coffee_amount?: number;
	time_seconds?: number;
	grind_size?: string;
	grinder_rkey?: string;
	brewer_rkey?: string;
	tasting_notes?: string;
	rating?: number;
	pours?: BrewPourInput[];
	espresso_params?: BrewEspressoParams;
	pourover_params?: BrewPouroverParams;
};

// BrewMutationResponse is the JSON envelope returned by POST /api/brews and
// PUT /api/brews/{id}. The brew record model carries no author field, so
// author_did is returned at the top level for SPA navigation. incomplete_nudge
// is populated when the referenced bean is missing fields.
export type BrewMutationResponse = {
	brew: Brew;
	author_did: string;
	incomplete_nudge?: {
		entity_type: string;
		rkey: string;
		name: string;
		missing: string;
	};
};

export function createBrew(fetchFn: typeof fetch, input: BrewInput): Promise<BrewMutationResponse> {
	return postJSON<BrewMutationResponse>(fetchFn, "/api/brews", input);
}

export function updateBrew(fetchFn: typeof fetch, rkey: string, input: BrewInput): Promise<BrewMutationResponse> {
	return putJSON<BrewMutationResponse>(fetchFn, `/api/brews/${encodeURIComponent(rkey)}`, input);
}

export function deleteEntity(fetchFn: typeof fetch, path: string): Promise<void> {
	return deleteJSON<void>(fetchFn, path);
}
