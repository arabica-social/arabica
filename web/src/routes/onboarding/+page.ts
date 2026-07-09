import type { PageLoad } from "./$types";
import { session } from "$lib/stores/session";
import { get } from "svelte/store";
import type { OnboardingResponse } from "$lib/types/api";

export const load: PageLoad = async ({ fetch }) => {
	const s = get(session);
	if (!s.isAuthenticated) {
		return { onboarding: null, error: "Authentication required" };
	}

	try {
		const res = await fetch("/api/onboarding", {
			headers: { Accept: "application/json" },
		});
		if (res.status === 401) {
			return { onboarding: null, error: "Authentication required" };
		}
		if (!res.ok) {
			return { onboarding: null, error: "Failed to load onboarding data" };
		}
		const onboarding = (await res.json()) as OnboardingResponse;
		return { onboarding, error: "" };
	} catch {
		return { onboarding: null, error: "Network error" };
	}
};
