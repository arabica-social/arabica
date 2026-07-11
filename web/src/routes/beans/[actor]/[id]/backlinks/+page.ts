import type { PageLoad } from "./$types";
import { fetchBacklinksView } from "$lib/api/backlinks";

export const load: PageLoad = async ({ fetch, params, url }) => {
	const usage = url.searchParams.get("usage") ?? "";
	const page = Number.parseInt(url.searchParams.get("page") ?? "", 10);
	return fetchBacklinksView(fetch, "beans", params.actor, params.id, {
		usage,
		page: Number.isNaN(page) ? 0 : page,
	});
};
