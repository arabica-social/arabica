import type { PageLoad } from "./$types";
import { get } from "svelte/store";
import { fetchEntityView } from "$lib/api/entityView";
import { session } from "$lib/stores/session";
import type { Roaster } from "$lib/types/entity_view";

export const load: PageLoad = async ({ fetch, params }) => {
	const current = get(session);
	if (!current.isAuthenticated) {
		return { roaster: null, error: "Authentication required" };
	}

	const actor = current.did || current.handle;
	const result = await fetchEntityView<Roaster>(fetch, "roasters", actor, params.id);
	return {
		roaster: result.data?.record ?? null,
		error: result.error ?? "",
	};
};
