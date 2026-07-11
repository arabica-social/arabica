import { redirect } from "@sveltejs/kit";
import type { PageLoad } from "./$types";

// /manage was the legacy brew-list page; it now lives at /my-coffee. The
// Go handler (HandleManage) still issues a 301 for non-SPA direct loads,
// but once the SPA shell owns the route the SvelteKit load function must
// perform the client-side redirect.
export const load: PageLoad = () => {
	redirect(301, "/my-coffee");
};
