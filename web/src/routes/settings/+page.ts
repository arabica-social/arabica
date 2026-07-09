import type { PageLoad } from "./$types";
import { session } from "$lib/stores/session";
import { get } from "svelte/store";
import type { SettingsResponse } from "$lib/types/api";

export const load: PageLoad = async ({ fetch }) => {
	const s = get(session);
	if (!s.isAuthenticated) {
		return { settings: null, error: "Authentication required" };
	}

	try {
		const res = await fetch("/api/settings", {
			headers: { Accept: "application/json" },
		});
		if (res.status === 401) {
			return { settings: null, error: "Authentication required" };
		}
		if (!res.ok) {
			return { settings: null, error: "Failed to load settings" };
		}
		const settings = (await res.json()) as SettingsResponse;
		return { settings, error: "" };
	} catch {
		return { settings: null, error: "Network error" };
	}
};
