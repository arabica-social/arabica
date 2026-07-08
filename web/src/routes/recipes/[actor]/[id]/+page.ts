import type { PageLoad } from "./$types";
import { fetchEntityView } from "$lib/api/entityView";
import type { Recipe } from "$lib/types/entity_view";

export const load: PageLoad = async ({ fetch, params }) => {
	const result = await fetchEntityView<Recipe>(
		fetch,
		"recipes",
		params.actor,
		params.id,
	);
	return result;
};
