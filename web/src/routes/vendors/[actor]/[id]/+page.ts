import type { PageLoad } from "./$types";
import { fetchEntityView } from "$lib/api/entityView";
import type { Vendor } from "$lib/types/generated/oolong_entities";

export const load: PageLoad = async ({ fetch, params }) => {
	return fetchEntityView<Vendor>(fetch, "vendors", params.actor, params.id);
};
