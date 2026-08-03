<script lang="ts">
	import { onMount } from "svelte";
	import Header from "$lib/components/Header.svelte";
	import Footer from "$lib/components/Footer.svelte";
	import LoginModal from "$lib/components/LoginModal.svelte";
	import { app, session, refreshSession } from "$lib/stores/session";
	import { themeStorageKey } from "$lib/stores/storageKeys";
	import {
		toasts,
		pushToast,
		dismissToast,
		extractNotifyMessage,
	} from "$lib/stores/toasts";
	import { appCache, installAppCacheGlobal } from "$lib/stores/appCache";
	import { clearFeedCache } from "$lib/stores/feedCache";
	import { definitionFor } from "$lib/app/definitions";

	let { children } = $props();

	let appDefinition = $derived(definitionFor($app));
	let brandName = $derived(appDefinition.displayName);
	let brandTagline = $derived(appDefinition.tagline);

	let showSessionModal = $state(false);
	let showLoginModal = $state(false);
	let sessionDialog = $state<HTMLDialogElement>();

	// Apply the saved theme and synchronize changes across tabs.
	// applyTheme reads the persisted theme choice and applies it to the
	// document. "system" (no stored value) removes the data-theme attribute
	// so CSS prefers-color-scheme drives light/dark. Exposed on window so
	// the Settings page theme buttons can trigger it after writing
	// localStorage.
	function applyTheme() {
		try {
			const theme = localStorage.getItem(themeStorageKey());
			if (theme === "dark" || theme === "light") {
				document.documentElement.setAttribute("data-theme", theme);
			} else {
				document.documentElement.removeAttribute("data-theme");
			}
		} catch {
			document.documentElement.removeAttribute("data-theme");
		}
		// Notify any listeners (e.g. settings page active-button refresh) that
		// the theme changed in this tab.
		document.body.dispatchEvent(new CustomEvent("arabica:theme-change"));
	}

	function handleNotify(event: Event) {
		const message = extractNotifyMessage((event as CustomEvent).detail);
		if (message) pushToast(message);
	}

	function handleRefreshManage() {
		appCache.invalidateCache();
		clearFeedCache();
	}

	function handleEntityDeleted() {
		appCache.invalidateCache();
		clearFeedCache();
	}

	function showSessionExpiredModal() {
		showSessionModal = true;
	}

	function dismissSessionModal() {
		showSessionModal = false;
	}

	function handleSessionBackdropClick(event: MouseEvent) {
		if (event.target === sessionDialog) {
			showSessionModal = false;
		}
	}

	// Drive the session-expired dialog via showModal()/close() (not the `open`
	// attribute) so it is promoted to the top layer: centered in the viewport
	// with a backdrop, matching LoginModal and the other dialogs.
	$effect(() => {
		const d = sessionDialog;
		if (!d) return;
		if (showSessionModal && !d.open) {
			d.showModal();
		} else if (!showSessionModal && d.open) {
			d.close();
		}
	});

	function showLoginModalFn() {
		showLoginModal = true;
	}

	function dismissLoginModal() {
		showLoginModal = false;
	}

	onMount(() => {
		// In production the Go SPA shell injects this state into <body>. Vite's
		// development shell cannot, so refreshSession uses the proxied API.
		void refreshSession();
	});

	$effect(() => {
		// Apply theme before first paint to prevent flash.
		applyTheme();

		// Entity controls share this cache through the global app contract.
		installAppCacheGlobal();

		// Preload the user's record cache on authed pages. Non-fatal if it
		// fails — components fall back to fetching per-route.
		if ($session.isAuthenticated) {
			void appCache.init();
		}

		window.__showSessionExpiredModal = showSessionExpiredModal;
		window.__showLoginModal = showLoginModalFn;
		window.applyTheme = applyTheme;
		window.addEventListener("notify", handleNotify);
		document.body.addEventListener("refreshManage", handleRefreshManage);
		document.body.addEventListener("entityDeleted", handleEntityDeleted);
		window.addEventListener("storage", applyTheme);

		return () => {
			if (window.__showSessionExpiredModal === showSessionExpiredModal) {
				delete window.__showSessionExpiredModal;
			}
			if (window.__showLoginModal === showLoginModalFn) {
				delete window.__showLoginModal;
			}
			window.removeEventListener("notify", handleNotify);
			document.body.removeEventListener("refreshManage", handleRefreshManage);
			document.body.removeEventListener("entityDeleted", handleEntityDeleted);
			window.removeEventListener("storage", applyTheme);
		};
	});
</script>

<svelte:head>
	<title>{brandName}</title>
</svelte:head>

<Header {brandName} />

<main class="grow container mx-auto py-8">
	{@render children()}
</main>

<Footer {brandName} tagline={brandTagline} />

<div id="toast-region" class="toast-region" aria-live="polite" aria-atomic="false">
	{#each $toasts as toast (toast.id)}
		<div class="toast" role="status">
			{toast.message}
		</div>
	{/each}
</div>

<dialog
	bind:this={sessionDialog}
	class="modal-dialog"
	data-testid="session-expired-modal"
	onclose={dismissSessionModal}
	onclick={handleSessionBackdropClick}
>
	<div class="modal-content text-center">
		<div class="mb-4">
			<svg class="w-12 h-12 mx-auto text-amber-600" fill="none" stroke="currentColor" stroke-width="1.5" viewBox="0 0 24 24">
				<path stroke-linecap="round" stroke-linejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 1 0-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 0 0 2.25-2.25v-6.75a2.25 2.25 0 0 0-2.25-2.25H6.75a2.25 2.25 0 0 0-2.25 2.25v6.75a2.25 2.25 0 0 0 2.25 2.25Z"></path>
			</svg>
		</div>
		<h3 class="modal-title text-center">Session Expired</h3>
		<p class="text-emphasis text-sm mb-6">
			Your login session has expired. Log back in to continue where you left off.
		</p>
		<div class="flex flex-col gap-3">
			<form id="reauth-form" method="POST" action="/reauth">
				{#if $session.handle}
					<input type="hidden" name="handle" value={$session.handle} />
				{/if}
				<input type="hidden" name="return_to" value={window.location.pathname} />
				<button type="submit" class="btn-primary w-full">Log In Again</button>
			</form>
			<button type="button" onclick={dismissSessionModal} class="btn-secondary w-full">
				Dismiss
			</button>
		</div>
	</div>
</dialog>

<LoginModal bind:open={showLoginModal} />
