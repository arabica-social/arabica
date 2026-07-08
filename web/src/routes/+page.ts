import type { PageLoad } from "./$types";

// Minimal load function proving the data-loading pipeline works. Reads
// the app identity from the <body> data attribute injected by the Go shell.
export const load: PageLoad = () => {
	const body = typeof document !== "undefined" ? document.body : null;
	return {
		app: body?.dataset.app ?? "arabica",
		userDID: body?.dataset.userDid ?? "",
	};
};
