import type { PageLoad } from "./$types";
import { session } from "$lib/stores/session";
import { get } from "svelte/store";
import type { OnboardingResponse } from "$lib/types/api";

export const load: PageLoad = async ({ fetch, url }) => {
	const s = get(session);
	if (!s.isAuthenticated) {
		return { onboarding: null, error: "Authentication required", mode: "library" as const, initialEntity: url.searchParams.get("entity") ?? "" };
	}

	try {
		const res = await fetch("/api/onboarding", {
			headers: { Accept: "application/json" },
		});
		if (res.status === 401) {
			return { onboarding: null, error: "Authentication required", mode: "library" as const, initialEntity: "" };
		}
		if (!res.ok) {
			return { onboarding: null, error: "Failed to load data", mode: "library" as const, initialEntity: "" };
		}
		const onboarding = (await res.json()) as OnboardingResponse;
		return { onboarding, error: "", mode: "library" as const, initialEntity: url.searchParams.get("entity") ?? "" };
	} catch {
		return { onboarding: null, error: "Network error", mode: "library" as const, initialEntity: "" };
	}
};
