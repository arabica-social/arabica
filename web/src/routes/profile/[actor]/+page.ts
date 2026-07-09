import type { PageLoad } from "./$types";
import { session } from "$lib/stores/session";
import { get } from "svelte/store";
import type { ProfileResponse } from "$lib/types/api";

export const load: PageLoad = async ({ fetch, params, url }) => {
	const s = get(session);
	const actor = params.actor;

	try {
		const offset = url.searchParams.get("brews_offset") ?? "0";
		const params_ = new URLSearchParams();
		params_.set("brews_offset", offset);
		params_.set("brews_limit", "25");

		const res = await fetch(`/api/profile/${encodeURIComponent(actor)}?${params_.toString()}`, {
			headers: { Accept: "application/json" },
		});

		if (res.status === 404) {
			return { profile: null, error: "Profile not found", actor };
		}
		if (!res.ok) {
			return { profile: null, error: `Failed to load profile (${res.status})`, actor };
		}

		const profile = (await res.json()) as ProfileResponse;
		return { profile, error: "", actor, isAuthenticated: s.isAuthenticated };
	} catch {
		return { profile: null, error: "Network error", actor, isAuthenticated: s.isAuthenticated };
	}
};
