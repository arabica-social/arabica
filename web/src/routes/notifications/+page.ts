import type { PageLoad } from "./$types";
import { session } from "$lib/stores/session";
import { get } from "svelte/store";
import type { NotificationsResponse } from "$lib/types/api";
import { APIError } from "$lib/api/client";
import { getNotifications } from "$lib/api/notifications";

export const load: PageLoad = async ({ fetch, url }) => {
	const s = get(session);
	if (!s.isAuthenticated) {
		return { notifications: null, nextCursor: "", error: "Authentication required" };
	}

	const cursor = url.searchParams.get("cursor") ?? "";
	try {
		const data: NotificationsResponse = await getNotifications(fetch, cursor);
		return {
			notifications: data.notifications ?? [],
			nextCursor: data.next_cursor ?? "",
			error: "",
		};
	} catch (error) {
		if (error instanceof APIError) {
			if (error.status === 401) {
				return { notifications: null, nextCursor: "", error: "Authentication required" };
			}
			if (error.kind !== "network") {
				return { notifications: null, nextCursor: "", error: "Failed to load notifications" };
			}
		}
		return { notifications: null, nextCursor: "", error: "Network error" };
	}
};
