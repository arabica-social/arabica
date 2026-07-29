import type { PageLoad } from "./$types";
import type { ManageResponseJSON, BrewListResponse } from "$lib/types/manage";

export const load: PageLoad = async ({ fetch }) => {
	// Fetch both authenticated datasets in parallel.
	try {
		const [manageRes, brewsRes] = await Promise.all([
			fetch("/api/manage", { headers: { Accept: "application/json" } }),
			fetch("/api/brews?limit=25", { headers: { Accept: "application/json" } }),
		]);

		if (manageRes.status === 401 || brewsRes.status === 401) {
			return { manage: null, brews: null, error: "Authentication required" };
		}

		let manage: ManageResponseJSON | null = null;
		let brews: BrewListResponse | null = null;
		let error = "";

		if (manageRes.ok) {
			manage = (await manageRes.json()) as ManageResponseJSON;
		} else {
			error = "Failed to load your records";
		}

		if (brewsRes.ok) {
			brews = (await brewsRes.json()) as BrewListResponse;
		}

		return { manage, brews, error };
	} catch {
		return { manage: null, brews: null, error: "Network error" };
	}
};
