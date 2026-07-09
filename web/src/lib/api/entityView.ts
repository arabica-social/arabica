import type { EntityViewResponse } from "$lib/types/entity_view";
import { APIError, requestJSON } from "./client";

/**
 * Fetches an entity view from the JSON API. Used by `+page.ts` load
 * functions for entity detail pages (roaster, grinder, brewer, bean,
 * brew, recipe).
 *
 * Returns the response on success, or an error object the page can
 * render. Never throws — load functions should not throw for 404s.
 */
export async function fetchEntityView<TRecord>(
	fetchFn: typeof fetch,
	entity: string,
	actor: string,
	id: string,
): Promise<{ data?: EntityViewResponse<TRecord>; error?: string; status: number }> {
	const url = `/api/${entity}/${encodeURIComponent(actor)}/${encodeURIComponent(id)}`;
	try {
		const data = await requestJSON<EntityViewResponse<TRecord>>(fetchFn, url);
		return { data, status: 200 };
	} catch (error) {
		if (error instanceof APIError) {
			if (error.status === 404) return { error: "Record not found", status: 404 };
			if (error.status === 401) return { error: "Authentication required", status: 401 };
			if (error.kind === "network") return { error: "Network error", status: 0 };
			return { error: error.message || `Failed to load (${error.status})`, status: error.status };
		}
		return { error: "Network error", status: 0 };
	}
}
