import type { EntityViewResponse } from "$lib/types/entity_view";

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
		const res = await fetchFn(url, {
			headers: { Accept: "application/json" },
		});
		if (!res.ok) {
			if (res.status === 404) {
				return { error: "Record not found", status: 404 };
			}
			if (res.status === 401) {
				return { error: "Authentication required", status: 401 };
			}
			return { error: `Failed to load (${res.status})`, status: res.status };
		}
		const data = (await res.json()) as EntityViewResponse<TRecord>;
		return { data, status: 200 };
	} catch {
		return { error: "Network error", status: 0 };
	}
}
