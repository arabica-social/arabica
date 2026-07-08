// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
import type { AppCacheAPI } from "$lib/types/appCache";

declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}

	interface Window {
		// Legacy bridge: existing Svelte islands (EntityCombo, BrewFormIsland,
		// …) read window.AppCache. The layout installs the appCache singleton
		// here during migration so islands keep working unchanged.
		AppCache?: AppCacheAPI;
		// Fetch helpers call this on 401 to show the session-expired modal.
		__showSessionExpiredModal?: () => void;
		// Theme toggle affordance (kept for parity with the legacy runtime).
		applyTheme?: () => void;
	}
}

export {};
