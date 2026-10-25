// Account-level brew preferences. Operational settings are owned by the server;
// local state is optimistic and is discarded whenever the session DID changes.
import { writable, derived, type Readable, type Writable } from "svelte/store";
import type { BrewFormMode, SettingsResponse } from "$lib/types/api";
import { session } from "./session";

export type UserPrefs = {
  smartAutofillEnabled: boolean;
  brew_form_mode: BrewFormMode;
  loading: boolean;
};

function defaults(): UserPrefs {
  return { smartAutofillEnabled: true, brew_form_mode: "quick", loading: true };
}

export const userPrefs: Writable<UserPrefs> = writable(defaults());

function accountState(did: string) {
  return {
    did,
    controller: new AbortController(),
    load: null as Promise<void> | null,
    write: null as Promise<void> | null,
    pending: null as BrewFormMode | null,
    modeRevision: 0,
    autofillRevision: 0,
  };
}
let account = accountState("");
session.subscribe(($session) => {
  const did = $session.isAuthenticated ? $session.did : "";
  if (did === account.did) return;
  account.controller.abort();
  account = accountState(did);
  userPrefs.set(defaults());
});

/** Updates the shared preference immediately after Settings saves it. */
export function setSmartAutofillEnabled(enabled: boolean) {
  account.autofillRevision++;
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

/** One load per account session; late reads cannot undo explicit changes. */
export function loadUserPrefs(): Promise<void> {
  const owner = account;
  if (owner.load) return owner.load;
  const modeRevision = owner.modeRevision;
  const autofillRevision = owner.autofillRevision;
  owner.load = (async () => {
    let preferences: SettingsResponse["user_preferences"] | undefined;
    try {
      if (owner.did) {
        const response = await fetch("/api/settings", {
          credentials: "same-origin",
          headers: { Accept: "application/json" },
          signal: owner.controller.signal,
        });
        if (!response.ok)
          throw new Error(`Settings request failed: ${response.status}`);
        preferences = ((await response.json()) as SettingsResponse)
          .user_preferences;
      }
    } catch (error) {
      if (owner === account)
        console.warn("Unable to load user preferences:", error);
    }
    if (owner !== account) return;
    userPrefs.update((prefs) => ({
      ...prefs,
      smartAutofillEnabled:
        owner.autofillRevision === autofillRevision
          ? preferences?.smart_autofill !== "off"
          : prefs.smartAutofillEnabled,
      brew_form_mode:
        owner.modeRevision === modeRevision && modeRevision === 0
          ? preferences?.brew_form_mode === "detailed"
            ? "detailed"
            : "quick"
          : prefs.brew_form_mode,
      loading: false,
    }));
  })();
  return owner.load;
}

/** Persist only explicit mode selections, without blocking or reverting a draft.
 * Writes are serialized and pending selections coalesced to the latest mode.
 */
export async function setBrewFormMode(mode: BrewFormMode): Promise<void> {
  const owner = account;
  owner.modeRevision++;
  userPrefs.update((prefs) => ({ ...prefs, brew_form_mode: mode }));
  if (!owner.did) return;
  owner.pending = mode;
  if (!owner.write) {
    owner.write = (async () => {
      while (owner === account && owner.pending !== null) {
        const next = owner.pending;
        owner.pending = null;
        try {
          const response = await fetch("/api/settings/preferences", {
            method: "POST",
            credentials: "same-origin",
            headers: {
              Accept: "application/json",
            },
            signal: owner.controller.signal,
            body: new URLSearchParams({ brew_form_mode: next }),
          });
          if (!response.ok)
            throw new Error(`Settings update failed: ${response.status}`);
        } catch (error) {
          if (owner === account)
            console.warn("Unable to save brew form mode:", error);
        }
      }
    })().finally(() => {
      owner.write = null;
    });
  }
  await owner.write;
}
