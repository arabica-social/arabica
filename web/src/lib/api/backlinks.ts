import type { BacklinksResult } from "$lib/types/entity_view";
import { APIError, requestJSON } from "./client";

export type BacklinksResponseJSON = {
	entity_noun: string;
	entity_name: string;
	back_url: string;
	detail_url: string;
	result: BacklinksResult | null;
};

/**
 * Fetches the community backlinks view for an entity record. Used by the
 * `+page.ts` load functions for the backlinks/community pages
 * (`/{entity}/{actor}/{id}/backlinks`).
 *
 * `entity` is the URL plural (beans, roasters, grinders, brewers, recipes).
 * `usage` and `page` are optional pagination params for a specific usage
 * group's "load more" (server-side paginated at 25 per page).
 *
 * Returns the response on success, or an error object the page can render.
 * Never throws — load functions should not throw for 404s/401s.
 */
export async function fetchBacklinksView(
	fetchFn: typeof fetch,
	entity: string,
	actor: string,
	id: string,
	opts: { usage?: string; page?: number } = {},
): Promise<{ data?: BacklinksResponseJSON; error?: string; status: number }> {
	const params = new URLSearchParams();
	if (opts.usage) params.set("usage", opts.usage);
	if (opts.page && opts.page > 0) params.set("page", String(opts.page));
	const query = params.toString();
	const url = `/api/${entity}/${encodeURIComponent(actor)}/${encodeURIComponent(id)}/backlinks${query ? `?${query}` : ""}`;
	try {
		const data = await requestJSON<BacklinksResponseJSON>(fetchFn, url);
		return { data, status: 200 };
	} catch (error) {
		if (error instanceof APIError) {
			if (error.status === 404) return { error: "Record not found", status: 404 };
			if (error.status === 401)
				return { error: "Authentication required", status: 401 };
			if (error.kind === "network") return { error: "Network error", status: 0 };
			return {
				error: error.message || `Failed to load (${error.status})`,
				status: error.status,
			};
		}
		return { error: "Network error", status: 0 };
	}
}
