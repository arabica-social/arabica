import type { PageLoad } from "./$types";
import { session, app } from "$lib/stores/session";
import { get } from "svelte/store";
import type { FeedResponse } from "$lib/types/feed";

export const load: PageLoad = async ({ fetch, url }) => {
	const s = get(session);
	const isAuthenticated = s.isAuthenticated;
	const appName = get(app);

	// Read filter/sort from the URL query so deep links and back-button work.
	const type = url.searchParams.get("type") ?? "";
	const sort = url.searchParams.get("sort") ?? "recent";

	let feed: FeedResponse | null = null;
	let error = "";

	// The feed is public (shows community activity). Authenticated viewers
	// get viewer-context fields (is_liked_by_viewer, is_owner).
	try {
		const params = new URLSearchParams();
		if (type) params.set("type", type);
		if (sort && sort !== "recent") params.set("sort", sort);
		const query = params.toString();
		const res = await fetch(`/api/feed${query ? `?${query}` : ""}`, {
			headers: { Accept: "application/json" },
		});
		if (res.ok) {
			feed = (await res.json()) as FeedResponse;
		} else {
			error = "Failed to load feed";
		}
	} catch {
		error = "Network error";
	}

	return {
		feed,
		error,
		typeFilter: type,
		sort,
		isAuthenticated,
		userDID: s.did,
		appName,
	};
};
