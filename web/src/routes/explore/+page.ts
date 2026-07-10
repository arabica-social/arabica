import type { PageLoad } from "./$types";
import { session } from "$lib/stores/session";
import { get } from "svelte/store";
import type { ExploreResponse } from "$lib/types/api";

export const load: PageLoad = async ({ fetch, url }) => {
	const s = get(session);

	// Read all filters from URL query params.
	const params = new URLSearchParams();
	const keys = ["type", "q", "sort", "cursor", "origin", "variety", "process", "roast_level", "roaster", "min_rating", "closed", "location", "grinder_type", "burr_type", "brewer_type", "ratio_min", "ratio_max"];
	for (const key of keys) {
		const val = url.searchParams.get(key);
		if (val) params.set(key, val);
	}

	try {
		const query = params.toString();
		const res = await fetch(`/api/explore${query ? `?${query}` : ""}`, {
			headers: { Accept: "application/json" },
		});
		if (res.status === 401) {
			return { explore: null, error: "Authentication required", isAuthenticated: false };
		}
		if (!res.ok) {
			return { explore: null, error: "Failed to load explore results", isAuthenticated: s.isAuthenticated };
		}
		const explore = (await res.json()) as ExploreResponse;
		return { explore, error: "", isAuthenticated: s.isAuthenticated };
	} catch {
		return { explore: null, error: "Network error", isAuthenticated: s.isAuthenticated };
	}
};
