import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

// Keep the historical /manage URL redirecting to the canonical collection route.
export const load: PageLoad = () => {
	redirect(301, "/my-coffee");
};
