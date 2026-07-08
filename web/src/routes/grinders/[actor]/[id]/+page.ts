import type { PageLoad } from "./$types";
import { fetchEntityView } from "$lib/api/entityView";
import type { Grinder } from "$lib/types/entity_view";

export const load: PageLoad = async ({ fetch, params }) => {
	const result = await fetchEntityView<Grinder>(
		fetch,
		"grinders",
		params.actor,
		params.id,
	);
	return result;
};
