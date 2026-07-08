import type { PageLoad } from "./$types";
import { fetchEntityView } from "$lib/api/entityView";
import type { Brew } from "$lib/types/entity_view";

export const load: PageLoad = async ({ fetch, params }) => {
	const result = await fetchEntityView<Brew>(
		fetch,
		"brews",
		params.actor,
		params.id,
	);
	return result;
};
