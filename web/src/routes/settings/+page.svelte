<script lang="ts">
	import { pushToast } from "$lib/stores/toasts";
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
	let tempUnit = $state(settings?.user_preferences.temperature_unit ?? "recorded");
	// svelte-ignore state_referenced_locally
	let beanAvgRating = $state(settings?.profile_stats_visibility.bean_avg_rating ?? "public");
	// svelte-ignore state_referenced_locally
	let roasterAvgRating = $state(settings?.profile_stats_visibility.roaster_avg_rating ?? "public");
	// svelte-ignore state_referenced_locally
	let bskyDisplayName = $state(settings?.bluesky_profile.display_name ?? "");
	let savingPrefs = $state(false);
	let savingVisibility = $state(false);
	let savingBsky = $state(false);

	$effect(() => {
		tempUnit = settings?.user_preferences.temperature_unit ?? "recorded";
		beanAvgRating = settings?.profile_stats_visibility.bean_avg_rating ?? "public";
		roasterAvgRating = settings?.profile_stats_visibility.roaster_avg_rating ?? "public";
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
			const value = localStorage.getItem("arabica-theme");
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
				localStorage.removeItem("arabica-theme");
			} else {
				localStorage.setItem("arabica-theme", value);
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

<div class="page-container-sm py-6">
	<h1 class="page-title mb-6">Settings</h1>

	{#if data.error}
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/login" class="btn-primary">Log In</a>
		</div>
	{:else if settings}
		<!-- Appearance -->
		<div class="card card-inner">
			<h2 class="text-lg font-semibold mb-4 text-primary">Appearance</h2>
			<span class="form-label">Theme</span>
			<p class="text-sm mb-3 text-muted">Choose how Arabica looks for you. System will follow your OS setting.</p>
			<div class="flex flex-wrap gap-2">
				<button type="button" class={theme === "system" ? "filter-pill-active" : "filter-pill"} aria-pressed={theme === "system"} onclick={() => setTheme("system")}>System</button>
				<button type="button" class={theme === "light" ? "filter-pill-active" : "filter-pill"} aria-pressed={theme === "light"} onclick={() => setTheme("light")}>Light</button>
				<button type="button" class={theme === "dark" ? "filter-pill-active" : "filter-pill"} aria-pressed={theme === "dark"} onclick={() => setTheme("dark")}>Dark</button>
			</div>
		</div>

		<!-- Brewing Preferences -->
		<div class="card card-inner mt-4">
			<h2 class="text-lg font-semibold mb-2 text-primary">Brewing Preferences</h2>
			<p class="text-sm mb-4 text-muted">These preferences are tied to your DID, so they follow you across devices. Theme stays device-local.</p>
			<form onsubmit={(e) => { e.preventDefault(); savePrefs(); }}>
				<label class="form-label" for="temp-unit">Preferred temperature unit</label>
				<select id="temp-unit" bind:value={tempUnit} class="form-select">
					<option value="recorded">Recorded units (default)</option>
					<option value="celsius">Celsius (°C)</option>
					<option value="fahrenheit">Fahrenheit (°F)</option>
				</select>
				<div class="mt-4 flex items-center gap-3">
					<button type="submit" class="btn-primary" disabled={savingPrefs}>
						{savingPrefs ? "Saving..." : "Save"}
					</button>
				</div>
			</form>
		</div>

		<!-- Profile Visibility -->
		<div class="card card-inner mt-4">
			<h2 class="text-lg font-semibold mb-2 text-primary">Profile Visibility</h2>
			<p class="text-sm mb-4 text-muted">Control which aggregate stats are visible to others on your profile page. These settings only affect what other people see — you always see your own stats.</p>
			<form onsubmit={(e) => { e.preventDefault(); saveVisibility(); }}>
				<div class="space-y-4">
					<div>
						<label class="form-label" for="bean-avg-rating">Bean average brew rating</label>
						<select id="bean-avg-rating" bind:value={beanAvgRating} class="form-select">
							<option value="public">Public</option>
							<option value="private">Only me</option>
						</select>
					</div>
					<div>
						<label class="form-label" for="roaster-avg-rating">Roaster average brew rating</label>
						<select id="roaster-avg-rating" bind:value={roasterAvgRating} class="form-select">
							<option value="public">Public</option>
							<option value="private">Only me</option>
						</select>
					</div>
				</div>
				<div class="mt-4 flex items-center gap-3">
					<button type="submit" class="btn-primary" disabled={savingVisibility}>
						{savingVisibility ? "Saving..." : "Save"}
					</button>
				</div>
			</form>
		</div>

		<!-- Bluesky Profile -->
		<div class="card card-inner mt-4">
			<h2 class="text-lg font-semibold mb-2 text-primary">Bluesky Profile</h2>
			<p class="text-sm mb-4 text-muted">
				Edit the display name and avatar on your Bluesky profile record. These changes write directly to your PDS and apply across every app that reads <code>app.bsky.actor.profile</code> — not just Arabica.
			</p>
			{#if bsky?.needs_auth_again}
				<p class="text-sm text-muted">Your session expired. <a class="link" href="/login">Sign in again</a> to continue.</p>
			{:else if !bsky?.has_scopes}
				<p class="text-sm mb-4 text-muted">
					Arabica didn't ask for permission to edit your Bluesky profile when you signed in. Granting it now means your PDS will prompt you to re-approve Arabica with a wider scope.
				</p>
				<form method="POST" action="/settings/bluesky-profile/upgrade-scopes">
					<input type="hidden" name="return_to" value="/settings" />
					<button type="submit" class="btn-primary">Grant permission to edit Bluesky profile</button>
				</form>
			{:else}
				{#if bsky?.load_error}
					<p class="text-sm mb-3" style="color: var(--danger, #c44);">{bsky.load_error}</p>
				{/if}
				<form onsubmit={saveBskyProfile} enctype="multipart/form-data">
					<div class="space-y-4">
						<div>
							<label class="form-label" for="bsky-display-name">Display name</label>
							<input id="bsky-display-name" type="text" name="displayName" class="form-input" maxlength="640" bind:value={bskyDisplayName} />
						</div>
						<div>
							<label class="form-label" for="bsky-avatar">Avatar</label>
							{#if bsky?.avatar_url}
								<div class="mb-2"><img src={bsky.avatar_url} alt="Current avatar" style="width: 64px; height: 64px; border-radius: 50%; object-fit: cover;" /></div>
							{/if}
							<input id="bsky-avatar" type="file" name="avatar" accept="image/*" class="form-file" />
							<p class="text-xs mt-1 text-muted">Optional. Leave empty to keep your current avatar. Max 1MB.</p>
						</div>
					</div>
					<div class="mt-4 flex items-center gap-3">
						<button type="submit" class="btn-primary" disabled={savingBsky}>
							{savingBsky ? "Saving..." : "Save Bluesky profile"}
						</button>
					</div>
				</form>
			{/if}
		</div>

		<!-- Developer -->
		<div class="card card-inner mt-4">
			<h2 class="text-lg font-semibold mb-2 text-primary">Developer</h2>
			<p class="text-sm mb-4 text-muted">Tools for inspecting AT Protocol data.</p>
			<label class="flex items-center gap-2 cursor-pointer">
				<input type="checkbox" class="form-checkbox" bind:checked={devMode} onchange={(e) => setDevMode((e.currentTarget as HTMLInputElement).checked)} />
				<span class="text-sm text-primary">Show "Copy AT URI" in action menus</span>
			</label>
		</div>
	{/if}
</div>
