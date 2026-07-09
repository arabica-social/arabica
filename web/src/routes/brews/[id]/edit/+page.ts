import type { PageLoad } from "./$types";
import { session } from "$lib/stores/session";
import { get } from "svelte/store";
import { fetchEntityView } from "$lib/api/entityView";
import type { Brew } from "$lib/types/entity_view";

export const load: PageLoad = async ({ fetch, params }) => {
	const s = get(session);
	if (!s.isAuthenticated) {
		return { brew: null, error: "Authentication required" };
	}

	// Edit: load the existing brew. The edit URL uses /brews/{id}/edit
	// (the owner is the current user). We fetch via the entity view API
	// with the user's own DID as the actor.
	const actor = s.did || s.handle;
	const result = await fetchEntityView<Brew>(fetch, "brews", actor, params.id);
	return {
		brew: result.data?.record ?? null,
		error: result.error ?? "",
	};
};
