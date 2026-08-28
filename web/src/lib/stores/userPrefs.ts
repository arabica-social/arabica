// Loads the account-level brew-form preferences (currently just
// smart_autofill) from GET /api/settings once per session. The brew form
// reads the preference from here instead of owning it; smart autofill
// defaults to enabled when the fetch fails or the key is absent.

import { writable, derived, type Readable, type Writable } from "svelte/store";
import type { SettingsResponse } from "$lib/types/api";

export type UserPrefs = {
  smartAutofillEnabled: boolean;
  loading: boolean;
};

function readUserPrefs(): UserPrefs {
  return { smartAutofillEnabled: true, loading: true };
}

export const userPrefs: Writable<UserPrefs> =
  writable<UserPrefs>(readUserPrefs());

/** Updates the shared preference immediately after Settings saves it. */
export function setSmartAutofillEnabled(enabled: boolean) {
  userPrefs.update((prefs) => ({ ...prefs, smartAutofillEnabled: enabled }));
}

export const smartAutofillEnabled: Readable<boolean> = derived(
  userPrefs,
  ($prefs) => $prefs.smartAutofillEnabled,
);

export const userPrefsLoading: Readable<boolean> = derived(
  userPrefs,
  ($prefs) => $prefs.loading,
);

/**
 * Fetches the smart_autofill preference from /api/settings. Missing keys and
 * any fetch failure resolve to the feature being enabled (the default).
 */
let userPrefsLoad: Promise<void> | null = null;

export function loadUserPrefs(): Promise<void> {
  if (userPrefsLoad) return userPrefsLoad;
  userPrefsLoad = (async () => {
    try {
      const response = await fetch("/api/settings", {
        credentials: "same-origin",
        headers: { Accept: "application/json" },
      });
      if (!response.ok) {
        throw new Error(`Settings request failed: ${response.status}`);
      }
      const data = (await response.json()) as SettingsResponse;
      userPrefs.set({
        smartAutofillEnabled: data.user_preferences.smart_autofill !== "off",
        loading: false,
      });
    } catch (error) {
      console.warn("Unable to load user preferences:", error);
      userPrefs.set({ smartAutofillEnabled: true, loading: false });
    }
  })();
  return userPrefsLoad;
}
