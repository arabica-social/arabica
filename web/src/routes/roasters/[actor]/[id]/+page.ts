import type { PageLoad } from "./$types";
import { fetchEntityView } from "$lib/api/entityView";
import type { Roaster } from "$lib/types/entity_view";

export const load: PageLoad = async ({ fetch, params }) => {
	const result = await fetchEntityView<Roaster>(
		fetch,
		"roasters",
		params.actor,
		params.id,
	);
	return result;
};
