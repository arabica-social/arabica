import type { PageLoad } from "./$types";
import { get } from "svelte/store";
import { session } from "$lib/stores/session";
import type { Recipe } from "$lib/types/entity_view";

export const load: PageLoad = async ({ fetch, url }) => {
	const s = get(session);
	if (!s.isAuthenticated) {
		return { recipes: [], error: "Authentication required", isAuthenticated: false };
	}

	// Seed the initial search from URL query params so deep links and the
	// back button preserve filter state.
	const params = new URLSearchParams();
	for (const key of ["q", "category", "brewer_type", "min_coffee", "max_coffee", "sort"]) {
		const val = url.searchParams.get(key);
		if (val) params.set(key, val);
	}

	try {
		const query = params.toString();
		const res = await fetch(`/api/recipes/suggestions${query ? `?${query}` : ""}`, {
			headers: { Accept: "application/json" },
		});
		if (res.status === 401) {
			return { recipes: [], error: "Authentication required", isAuthenticated: false };
		}
		if (!res.ok) {
			return { recipes: [], error: "Failed to load recipes", isAuthenticated: true };
		}
		const recipes = (await res.json()) as Recipe[];
		return { recipes, error: "", isAuthenticated: true };
	} catch {
		return { recipes: [], error: "Network error", isAuthenticated: true };
	}
};
