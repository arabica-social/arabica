import type { PageLoad } from "./$types";
import type { BrewListResponse } from "$lib/types/manage";
import { requestJSON, APIError } from "$lib/api/client";

export const load: PageLoad = async ({ fetch }) => {
	// The brew list requires authentication; a 401 redirects to login.
	try {
		const data = await requestJSON<BrewListResponse>(fetch, "/api/brews?limit=25");
		return { brews: data, error: "" };
	} catch (error) {
		if (error instanceof APIError && error.status === 401) {
			return { brews: null, error: "Authentication required" };
		}
		const msg =
			error instanceof APIError && error.kind === "network"
				? "Network error"
				: "Failed to load brews";
		return { brews: null, error: msg };
	}
};
