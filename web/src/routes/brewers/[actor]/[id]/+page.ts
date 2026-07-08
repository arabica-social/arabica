import type { PageLoad } from "./$types";
import { fetchEntityView } from "$lib/api/entityView";
import type { Brewer } from "$lib/types/entity_view";

export const load: PageLoad = async ({ fetch, params }) => {
	const result = await fetchEntityView<Brewer>(
		fetch,
		"brewers",
		params.actor,
		params.id,
	);
	return result;
};
