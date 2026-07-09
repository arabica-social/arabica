import type { PageLoad } from "./$types";
import { session } from "$lib/stores/session";
import { get } from "svelte/store";
import type { Brew } from "$lib/types/entity_view";

export const load: PageLoad = async ({ fetch, url }) => {
	const s = get(session);
	if (!s.isAuthenticated) {
		return { brew: null, error: "Authentication required", recipeRKey: "", recipeOwnerDID: "" };
	}

	const recipeRKey = url.searchParams.get("recipe") ?? "";
	const recipeOwnerDID = url.searchParams.get("recipe_owner") ?? "";

	// New brew: no existing record to load.
	return {
		brew: null as Brew | null,
		error: "",
		recipeRKey,
		recipeOwnerDID,
	};
};
