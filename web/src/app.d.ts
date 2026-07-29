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
    // Shared entity controls access the record cache through this global contract.
		AppCache?: AppCacheAPI;
		// Fetch helpers call this on 401 to show the session-expired modal.
		__showSessionExpiredModal?: () => void;
    // Applies the persisted theme choice.
		applyTheme?: () => void;
	}
}

export {};
