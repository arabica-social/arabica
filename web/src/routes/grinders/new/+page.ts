import type { PageLoad } from "./$types";
import { get } from "svelte/store";
import { session } from "$lib/stores/session";

export const load: PageLoad = async () => {
	if (!get(session).isAuthenticated) {
		return { error: "Authentication required" };
	}
	return { error: "" };
};
