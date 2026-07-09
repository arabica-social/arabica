import { requestJSON } from "./client";
import type { NotificationsResponse } from "$lib/types/api";

export function getNotifications(
	fetchFn: typeof fetch,
	cursor = "",
): Promise<NotificationsResponse> {
	const params = new URLSearchParams();
	if (cursor) params.set("cursor", cursor);
	const query = params.toString();
	return requestJSON<NotificationsResponse>(
		fetchFn,
		`/api/notifications${query ? `?${query}` : ""}`,
	);
}

export function markAllNotificationsRead(fetchFn: typeof fetch): Promise<{ read: boolean }> {
	return requestJSON<{ read: boolean }>(fetchFn, "/api/notifications/read", {
		method: "POST",
	});
}
