<script lang="ts">
  import { pushToast } from "$lib/stores/toasts";
  import { openLoginModal } from "$lib/stores/session";
  import { themeStorageKey } from "$lib/stores/storageKeys";
  import LedgerHeader from "$lib/components/LedgerHeader.svelte";
  import type { PageData } from "./$types";
  import type { SettingsResponse } from "$lib/types/api";

  let { data }: { data: PageData } = $props();

  // svelte-ignore state_referenced_locally
  let settings = $state<SettingsResponse | null>(data.settings);

  // svelte-ignore state_referenced_locally
  $effect(() => {
    settings = data.settings;
  });

  // svelte-ignore state_referenced_locally
  let tempUnit = $state(
    settings?.user_preferences.temperature_unit ?? "recorded",
  );
  // svelte-ignore state_referenced_locally
  let beanAvgRating = $state(
    settings?.profile_stats_visibility.bean_avg_rating ?? "public",
  );
  // svelte-ignore state_referenced_locally
  let roasterAvgRating = $state(
    settings?.profile_stats_visibility.roaster_avg_rating ?? "public",
  );
  // svelte-ignore state_referenced_locally
  let bskyDisplayName = $state(settings?.bluesky_profile.display_name ?? "");
  let savingPrefs = $state(false);
  let savingVisibility = $state(false);
  let savingBsky = $state(false);

  $effect(() => {
    tempUnit = settings?.user_preferences.temperature_unit ?? "recorded";
    beanAvgRating =
      settings?.profile_stats_visibility.bean_avg_rating ?? "public";
    roasterAvgRating =
      settings?.profile_stats_visibility.roaster_avg_rating ?? "public";
    bskyDisplayName = settings?.bluesky_profile.display_name ?? "";
  });

  async function savePrefs() {
    savingPrefs = true;
    try {
      const res = await fetch("/api/settings/preferences", {
        method: "POST",
        credentials: "same-origin",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          Accept: "application/json",
        },
        body: new URLSearchParams({ temperature_unit: tempUnit }),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      pushToast("Preferences saved");
    } catch {
      pushToast("Failed to save preferences");
    } finally {
      savingPrefs = false;
    }
  }

  async function saveVisibility() {
    savingVisibility = true;
    try {
      const res = await fetch("/api/settings/profile-visibility", {
        method: "POST",
        credentials: "same-origin",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
          Accept: "application/json",
        },
        body: new URLSearchParams({
          bean_avg_rating: beanAvgRating,
          roaster_avg_rating: roasterAvgRating,
        }),
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      pushToast("Visibility saved");
    } catch {
      pushToast("Failed to save visibility");
    } finally {
      savingVisibility = false;
    }
  }

  async function saveBskyProfile(e: SubmitEvent) {
    e.preventDefault();
    savingBsky = true;
    try {
      const form = e.target as HTMLFormElement;
      const formData = new FormData(form);
      const res = await fetch("/api/settings/bluesky-profile", {
        method: "POST",
        credentials: "same-origin",
        headers: { Accept: "application/json" },
        body: formData,
      });
      if (!res.ok) throw new Error(`Failed: ${res.status}`);
      pushToast("Bluesky profile saved");
    } catch {
      pushToast("Failed to save Bluesky profile");
    } finally {
      savingBsky = false;
    }
  }

  let bsky = $derived(settings?.bluesky_profile);

  // --- Appearance: theme + developer mode ---
  type Theme = "system" | "light" | "dark";
  function readTheme(): Theme {
    try {
      const value = localStorage.getItem(themeStorageKey());
      return value === "light" || value === "dark" ? value : "system";
    } catch {
      return "system";
    }
  }
  let theme = $state<Theme>("system");
  let devMode = $state(false);

  function setTheme(value: Theme) {
    theme = value;
    try {
      if (value === "system") {
        localStorage.removeItem(themeStorageKey());
      } else {
        localStorage.setItem(themeStorageKey(), value);
      }
    } catch {
      // localStorage may be unavailable in strict privacy modes.
    }
    window.applyTheme?.();
  }

  function setDevMode(value: boolean) {
    devMode = value;
    try {
      localStorage.setItem("devMode", String(value));
    } catch {
      // localStorage may be unavailable in strict privacy modes.
    }
    window.dispatchEvent(new CustomEvent("arabica:dev-mode-change"));
  }

  $effect(() => {
    theme = readTheme();
    try {
      devMode = localStorage.getItem("devMode") === "true";
    } catch {
      devMode = false;
    }
    const onStorage = () => {
      theme = readTheme();
      try {
        devMode = localStorage.getItem("devMode") === "true";
      } catch {
        devMode = false;
      }
    };
    const onThemeChange = () => {
      theme = readTheme();
    };
    window.addEventListener("storage", onStorage);
    document.body.addEventListener("arabica:theme-change", onThemeChange);
    return () => {
      window.removeEventListener("storage", onStorage);
      document.body.removeEventListener("arabica:theme-change", onThemeChange);
    };
  });
</script>

<svelte:head>
  <title>Settings - Arabica</title>
</svelte:head>

<div class="settings-ledger">
  <LedgerHeader
    title="Settings"
    eyebrow="Account ledger"
    description="Choose how your journal looks, how measurements are displayed, and what your public profile shares."
  />

  {#if data.error}
    <div class="settings-state">
      <p>{data.error}</p>
      <button type="button" onclick={openLoginModal} class="btn-primary"
        >Log In</button
      >
    </div>
  {:else if settings}
    <div class="settings-workspace">
      <nav class="settings-index" aria-label="Settings sections">
        <p class="settings-index__label">Index</p>
        <a href="#appearance"><span>01</span>Appearance</a>
        <a href="#brewing"><span>02</span>Brewing</a>
        <a href="#visibility"><span>03</span>Visibility</a>
        <a href="#bluesky"><span>04</span>Bluesky profile</a>
        <a href="#developer"><span>05</span>Developer</a>
      </nav>

      <div class="settings-sheet">
        <section
          id="appearance"
          class="settings-section"
          aria-labelledby="appearance-title"
        >
          <header class="settings-section__header">
            <span class="settings-section__number">01</span>
            <div>
              <p class="settings-section__eyebrow">This device</p>
              <h2 id="appearance-title">Appearance</h2>
            </div>
          </header>
          <p class="settings-section__intro">
            System follows your operating system. Light and dark stay local to
            this device.
          </p>
          <div class="theme-choices" role="group" aria-label="Theme">
            <button
              type="button"
              class="theme-choice"
              aria-pressed={theme === "system"}
              onclick={() => setTheme("system")}
            >
              <span
                class="theme-preview theme-preview--system"
                aria-hidden="true"><i></i><i></i></span
              >
              <span
                ><strong>System</strong><small>Follow this device</small></span
              >
            </button>
            <button
              type="button"
              class="theme-choice"
              aria-pressed={theme === "light"}
              onclick={() => setTheme("light")}
            >
              <span
                class="theme-preview theme-preview--light"
                aria-hidden="true"><i></i></span
              >
              <span><strong>Light</strong><small>Cream paper</small></span>
            </button>
            <button
              type="button"
              class="theme-choice"
              aria-pressed={theme === "dark"}
              onclick={() => setTheme("dark")}
            >
              <span class="theme-preview theme-preview--dark" aria-hidden="true"
                ><i></i></span
              >
              <span><strong>Dark</strong><small>Espresso paper</small></span>
            </button>
          </div>
        </section>

        <section
          id="brewing"
          class="settings-section"
          aria-labelledby="brewing-title"
        >
          <header class="settings-section__header">
            <span class="settings-section__number">02</span>
            <div>
              <p class="settings-section__eyebrow">Travels with your DID</p>
              <h2 id="brewing-title">Brewing preferences</h2>
            </div>
          </header>
          <form
            class="settings-form"
            onsubmit={(e) => {
              e.preventDefault();
              savePrefs();
            }}
          >
            <div class="settings-row">
              <div>
                <label for="temp-unit">Preferred temperature unit</label>
                <p>
                  Display recorded values as entered, or convert them throughout
                  your journal.
                </p>
              </div>
              <select id="temp-unit" bind:value={tempUnit} class="form-select">
                <option value="recorded">Recorded units</option>
                <option value="celsius">Celsius (°C)</option>
                <option value="fahrenheit">Fahrenheit (°F)</option>
              </select>
            </div>
            <div class="settings-form__actions">
              <button type="submit" class="btn-primary" disabled={savingPrefs}
                >{savingPrefs ? "Saving..." : "Save"}</button
              >
            </div>
          </form>
        </section>

        <section
          id="visibility"
          class="settings-section"
          aria-labelledby="visibility-title"
        >
          <header class="settings-section__header">
            <span class="settings-section__number">03</span>
            <div>
              <p class="settings-section__eyebrow">Public profile</p>
              <h2 id="visibility-title">Profile visibility</h2>
            </div>
          </header>
          <p class="settings-section__intro">
            These choices affect visitors to your profile. You always see your
            own statistics.
          </p>
          <form
            class="settings-form"
            onsubmit={(e) => {
              e.preventDefault();
              saveVisibility();
            }}
          >
            <div class="settings-row">
              <div>
                <label for="bean-avg-rating">Bean average brew rating</label>
                <p>
                  Show the average rating calculated from your brews for each
                  bean.
                </p>
              </div>
              <select
                id="bean-avg-rating"
                bind:value={beanAvgRating}
                class="form-select"
              >
                <option value="public">Public</option>
                <option value="private">Only me</option>
              </select>
            </div>
            <div class="settings-row">
              <div>
                <label for="roaster-avg-rating"
                  >Roaster average brew rating</label
                >
                <p>
                  Show the average rating calculated across beans from each
                  roaster.
                </p>
              </div>
              <select
                id="roaster-avg-rating"
                bind:value={roasterAvgRating}
                class="form-select"
              >
                <option value="public">Public</option>
                <option value="private">Only me</option>
              </select>
            </div>
            <div class="settings-form__actions">
              <button
                type="submit"
                class="btn-primary"
                disabled={savingVisibility}
                >{savingVisibility ? "Saving..." : "Save"}</button
              >
            </div>
          </form>
        </section>

        <section
          id="bluesky"
          class="settings-section"
          aria-labelledby="bluesky-title"
        >
          <header class="settings-section__header">
            <span class="settings-section__number">04</span>
            <div>
              <p class="settings-section__eyebrow">Shared identity</p>
              <h2 id="bluesky-title">Bluesky profile</h2>
            </div>
          </header>
          <p class="settings-section__intro">
            Changes write directly to your PDS and apply to every app that reads <code
              >app.bsky.actor.profile</code
            >, not only Arabica.
          </p>
          {#if bsky?.needs_auth_again}
            <p class="settings-notice">
              Your session expired. <a class="link" href="/login"
                >Sign in again</a
              > to continue.
            </p>
          {:else if !bsky?.has_scopes}
            <p class="settings-notice">
              Editing this shared profile requires a wider OAuth scope. Your PDS
              will ask you to approve it.
            </p>
            <form
              method="POST"
              action="/settings/bluesky-profile/upgrade-scopes"
            >
              <input type="hidden" name="return_to" value="/settings" />
              <button type="submit" class="btn-primary"
                >Grant profile permission</button
              >
            </form>
          {:else}
            {#if bsky?.load_error}<p class="settings-error">
                {bsky.load_error}
              </p>{/if}
            <form
              class="settings-form"
              onsubmit={saveBskyProfile}
              enctype="multipart/form-data"
            >
              <div class="settings-profile-grid">
                {#if bsky?.avatar_url}<img
                    src={bsky.avatar_url}
                    alt="Current avatar"
                    class="settings-avatar"
                  />{/if}
                <div>
                  <label for="bsky-display-name">Display name</label>
                  <input
                    id="bsky-display-name"
                    type="text"
                    name="displayName"
                    class="form-input"
                    maxlength="640"
                    bind:value={bskyDisplayName}
                  />
                </div>
              </div>
              <div class="settings-row settings-row--file">
                <div>
                  <label for="bsky-avatar">Avatar</label>
                  <p>
                    Leave this empty to keep the current image. Maximum size: 1
                    MB.
                  </p>
                </div>
                <input
                  id="bsky-avatar"
                  type="file"
                  name="avatar"
                  accept="image/*"
                  class="form-file"
                />
              </div>
              <div class="settings-form__actions">
                <button type="submit" class="btn-primary" disabled={savingBsky}
                  >{savingBsky ? "Saving..." : "Save Bluesky profile"}</button
                >
              </div>
            </form>
          {/if}
        </section>

        <section
          id="developer"
          class="settings-section settings-section--quiet"
          aria-labelledby="developer-title"
        >
          <header class="settings-section__header">
            <span class="settings-section__number">05</span>
            <div>
              <p class="settings-section__eyebrow">Advanced</p>
              <h2 id="developer-title">Developer</h2>
            </div>
          </header>
          <label class="settings-check">
            <input
              type="checkbox"
              class="form-checkbox"
              bind:checked={devMode}
              onchange={(e) =>
                setDevMode((e.currentTarget as HTMLInputElement).checked)}
            />
            <span
              ><strong>Show “Copy AT URI” in action menus</strong><small
                >Useful when inspecting records directly on AT Protocol.</small
              ></span
            >
          </label>
        </section>
      </div>
    </div>
  {/if}
</div>

<style>
  .settings-ledger {
    width: 100%;
    max-width: 72rem;
    margin-inline: auto;
    padding: 0.5rem clamp(0.5rem, 2.2vw, 2rem) 3rem;
  }
  .settings-workspace {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1.5rem;
    padding-top: 1.5rem;
  }
  .settings-index {
    display: flex;
    gap: 0.35rem;
    overflow-x: auto;
    padding: 0 0 0.75rem;
    border-bottom: 1px solid var(--card-border);
    scrollbar-width: thin;
  }
  .settings-index__label {
    display: none;
  }
  .settings-index a {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    min-height: 2.75rem;
    padding: 0.5rem 0.75rem;
    border: 1px solid var(--card-border);
    border-radius: 0.4rem;
    color: var(--text-muted);
    font-size: 0.76rem;
    font-weight: 600;
    text-decoration: none;
    white-space: nowrap;
  }
  .settings-index a span {
    color: var(--text-faint);
    font-size: 0.62rem;
    letter-spacing: 0.08em;
  }
  .settings-index a:hover,
  .settings-index a:focus-visible {
    color: var(--text-primary);
    border-color: var(--text-faint);
    background: var(--surface-bg);
  }
  .settings-sheet {
    min-width: 0;
    border: 1px solid var(--card-border);
    border-radius: 0.75rem;
    background: var(--card-bg);
    background-image: var(--texture-kraft);
    background-blend-mode: multiply;
    box-shadow: var(--shadow-sm);
    overflow: clip;
  }
  .settings-section {
    scroll-margin-top: 6rem;
    padding: 1.5rem clamp(1rem, 3vw, 2rem) 1.75rem;
    border-top: 1px solid var(--card-border);
  }
  .settings-section:first-child {
    border-top: 3px double var(--text-primary);
  }
  .settings-section--quiet {
    background: color-mix(in oklch, var(--surface-bg) 70%, transparent);
  }
  .settings-section__header {
    display: flex;
    align-items: flex-start;
    gap: 0.85rem;
  }
  .settings-section__number {
    flex: 0 0 auto;
    color: var(--text-faint);
    font-size: 0.68rem;
    font-weight: 600;
    letter-spacing: 0.12em;
    padding-top: 0.35rem;
  }
  .settings-section__eyebrow {
    margin: 0 0 0.15rem;
    color: var(--text-faint);
    font-size: 0.64rem;
    font-weight: 600;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .settings-section h2 {
    margin: 0;
    color: var(--text-primary);
    font-family: var(--font-display);
    font-size: 1.45rem;
    font-weight: 600;
    letter-spacing: -0.015em;
  }
  .settings-section__intro {
    max-width: 64ch;
    margin: 0.85rem 0 1.25rem 2.25rem;
    color: var(--text-muted);
    font-size: 0.88rem;
    line-height: 1.6;
  }
  .theme-choices {
    display: grid;
    grid-template-columns: 1fr;
    gap: 0.75rem;
    margin-left: 2.25rem;
  }
  .theme-choice {
    display: grid;
    grid-template-columns: 4.5rem 1fr;
    align-items: center;
    gap: 0.85rem;
    min-height: 5rem;
    padding: 0.65rem;
    border: 1px solid var(--card-border);
    border-radius: 0.55rem;
    color: var(--text-primary);
    background: color-mix(in oklch, var(--card-bg) 88%, transparent);
    text-align: left;
    transition:
      border-color 150ms ease-out,
      transform 150ms ease-out,
      background 150ms ease-out;
  }
  .theme-choice:hover {
    transform: translateY(-1px);
    border-color: var(--text-faint);
  }
  .theme-choice[aria-pressed="true"] {
    border-color: var(--header-bg);
    background: color-mix(in oklch, var(--type-bean-tint) 45%, var(--card-bg));
    box-shadow: inset 0 0 0 1px var(--header-bg);
  }
  .theme-choice strong,
  .theme-choice small {
    display: block;
  }
  .theme-choice strong {
    font-size: 0.9rem;
  }
  .theme-choice small {
    margin-top: 0.15rem;
    color: var(--text-muted);
    font-size: 0.72rem;
  }
  .theme-preview {
    display: flex;
    width: 4.5rem;
    height: 3.25rem;
    overflow: hidden;
    border: 1px solid var(--card-border);
    border-radius: 0.35rem;
    box-shadow: var(--shadow-sm);
  }
  .theme-preview i {
    flex: 1;
    position: relative;
  }
  .theme-preview i::after {
    content: "";
    position: absolute;
    inset: 0.55rem 0.4rem auto;
    height: 1px;
    background: currentColor;
    box-shadow:
      0 0.45rem 0 currentColor,
      0 0.9rem 0 currentColor;
    opacity: 0.45;
  }
  .theme-preview--system i:first-child,
  .theme-preview--light i {
    color: #4a2c2a;
    background: #faf7f5;
  }
  .theme-preview--system i:last-child,
  .theme-preview--dark i {
    color: #f2e8e5;
    background: #2b1b18;
  }
  .settings-form {
    margin: 1.25rem 0 0 2.25rem;
  }
  .settings-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 0.75rem;
    align-items: center;
    padding: 1rem 0;
    border-top: 1px solid var(--surface-border);
  }
  .settings-row:first-child {
    border-top: 0;
    padding-top: 0;
  }
  .settings-row label,
  .settings-profile-grid label {
    display: block;
    color: var(--text-primary);
    font-size: 0.86rem;
    font-weight: 600;
  }
  .settings-row p {
    max-width: 55ch;
    margin: 0.25rem 0 0;
    color: var(--text-muted);
    font-size: 0.76rem;
    line-height: 1.5;
  }
  .settings-row .form-select {
    width: 100%;
  }
  .settings-form__actions {
    display: flex;
    justify-content: flex-end;
    padding-top: 1rem;
    border-top: 1px solid var(--surface-border);
  }
  .settings-profile-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1rem;
    align-items: end;
    padding-bottom: 1rem;
  }
  .settings-profile-grid .form-input {
    width: 100%;
    margin-top: 0.4rem;
  }
  .settings-avatar {
    width: 4.5rem;
    height: 4.5rem;
    border: 2px solid var(--card-bg);
    border-radius: 50%;
    object-fit: cover;
    box-shadow:
      0 0 0 1px var(--card-border),
      var(--shadow-sm);
  }
  .settings-notice,
  .settings-error {
    margin: 1rem 0 0 2.25rem;
    padding: 0.85rem 1rem;
    border: 1px solid var(--card-border);
    border-radius: 0.4rem;
    background: var(--surface-bg);
    color: var(--text-muted);
    font-size: 0.82rem;
    line-height: 1.5;
  }
  .settings-error {
    color: var(--brand-red-600);
    border-color: color-mix(
      in oklch,
      var(--brand-red-600) 30%,
      var(--card-border)
    );
  }
  .settings-check {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    margin: 1rem 0 0 2.25rem;
    cursor: pointer;
  }
  .settings-check input {
    margin-top: 0.2rem;
  }
  .settings-check strong,
  .settings-check small {
    display: block;
  }
  .settings-check strong {
    color: var(--text-primary);
    font-size: 0.86rem;
  }
  .settings-check small {
    max-width: 55ch;
    margin-top: 0.2rem;
    color: var(--text-muted);
    font-size: 0.76rem;
    line-height: 1.5;
  }
  .settings-state {
    margin-top: 1.5rem;
    padding: 3rem 1rem;
    border: 1px solid var(--card-border);
    border-radius: 0.75rem;
    background: var(--card-bg);
    text-align: center;
  }
  .settings-state p {
    margin: 0 0 1rem;
    color: var(--text-secondary);
  }
  @media (min-width: 640px) {
    .theme-choices {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
    .theme-choice {
      grid-template-columns: 1fr;
    }
    .settings-row {
      grid-template-columns: minmax(0, 1fr) minmax(11rem, 0.42fr);
      gap: 1.5rem;
    }
    .settings-profile-grid {
      grid-template-columns: auto minmax(0, 1fr);
    }
  }
  @media (min-width: 900px) {
    .settings-workspace {
      grid-template-columns: 12rem minmax(0, 1fr);
      gap: clamp(1.5rem, 3vw, 2.5rem);
      align-items: start;
    }
    .settings-index {
      position: sticky;
      top: 5.5rem;
      display: flex;
      flex-direction: column;
      gap: 0;
      overflow: visible;
      padding: 0.75rem 0;
      border-top: 2px solid var(--text-primary);
      border-bottom: 1px solid var(--card-border);
    }
    .settings-index__label {
      display: block;
      margin: 0 0 0.5rem;
      padding: 0 0.5rem;
      color: var(--text-faint);
      font-size: 0.64rem;
      font-weight: 600;
      letter-spacing: 0.14em;
      text-transform: uppercase;
    }
    .settings-index a {
      min-height: 2.6rem;
      border: 0;
      border-top: 1px solid var(--surface-border);
      border-radius: 0;
      padding-inline: 0.5rem;
    }
    .settings-index a:last-child {
      border-bottom: 1px solid var(--surface-border);
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .theme-choice {
      transition: none;
    }
  }
</style>
