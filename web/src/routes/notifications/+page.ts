import type { PageLoad } from "./$types";
import { session } from "$lib/stores/session";
import { get } from "svelte/store";
import type { NotificationsResponse } from "$lib/types/api";

export const load: PageLoad = async ({ fetch, url }) => {
	const s = get(session);
	if (!s.isAuthenticated) {
		return { notifications: null, nextCursor: "", error: "Authentication required" };
	}

	const cursor = url.searchParams.get("cursor") ?? "";
	try {
		const params = new URLSearchParams();
		if (cursor) params.set("cursor", cursor);
		const query = params.toString();
		const res = await fetch(`/api/notifications${query ? `?${query}` : ""}`, {
			headers: { Accept: "application/json" },
		});
		if (res.status === 401) {
			return { notifications: null, nextCursor: "", error: "Authentication required" };
		}
		if (!res.ok) {
			return { notifications: null, nextCursor: "", error: "Failed to load notifications" };
		}
		const data = (await res.json()) as NotificationsResponse;
		return {
			notifications: data.notifications ?? [],
			nextCursor: data.next_cursor ?? "",
			error: "",
		};
	} catch {
		return { notifications: null, nextCursor: "", error: "Network error" };
	}
};
