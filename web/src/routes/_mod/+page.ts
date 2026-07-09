import type { PageLoad } from "./$types";
import { session } from "$lib/stores/session";
import { get } from "svelte/store";
import type { AdminResponse } from "$lib/types/api";

export const load: PageLoad = async ({ fetch }) => {
	const s = get(session);
	if (!s.isAuthenticated) {
		return { admin: null, error: "Authentication required" };
	}

	try {
		const res = await fetch("/api/_mod", {
			headers: { Accept: "application/json" },
		});
		if (res.status === 403) {
			return { admin: null, error: "You don't have permission to access the moderation dashboard." };
		}
		if (res.status === 401) {
			return { admin: null, error: "Authentication required" };
		}
		if (!res.ok) {
			return { admin: null, error: `Failed to load admin data (${res.status})` };
		}
		const admin = (await res.json()) as AdminResponse;
		return { admin, error: "" };
	} catch {
		return { admin: null, error: "Network error" };
	}
};
