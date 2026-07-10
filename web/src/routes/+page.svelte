<script lang="ts">
	import FeedCard from "$lib/components/FeedCard.svelte";
	import FeedFilters from "$lib/components/FeedFilters.svelte";
	import Icon from "$lib/components/Icon.svelte";
	import { session } from "$lib/stores/session";
	import { pushToast } from "$lib/stores/toasts";
	import { goto } from "$app/navigation";
	import { tick } from "svelte";
	import { applyFeedMasonry, installFeedMasonry } from "$lib/utils/feedMasonry";
	import type { PageData } from "./$types";
	import type { FeedItem, FeedResponse } from "$lib/types/feed";

	let { data }: { data: PageData } = $props();

	let s = $derived(data);

	// svelte-ignore state_referenced_locally
	let typeFilter = $state(data.typeFilter ?? "");
	// svelte-ignore state_referenced_locally
	let sort = $state(data.sort ?? "recent");
	// svelte-ignore state_referenced_locally
	let items = $state<FeedItem[]>(data.feed?.items ?? []);
	// svelte-ignore state_referenced_locally
	let nextCursor = $state(data.feed?.next_cursor ?? "");
	let loading = $state(false);
	let loadingMore = $state(false);

	// Sync load function data into local state.
	// svelte-ignore state_referenced_locally
	$effect(() => {
		typeFilter = data.typeFilter ?? "";
		sort = data.sort ?? "recent";
		items = data.feed?.items ?? [];
		nextCursor = data.feed?.next_cursor ?? "";
	});

	// Feed masonry: re-pack cards into two corkboard columns whenever the
	// item set changes (filter/sort/load-more). `applyFeedMasonry` measures
	// offsetHeight, so it runs after Svelte commits the DOM (tick) and after
	// the browser paints (requestAnimationFrame).
	$effect(() => {
		// Reference items so the effect re-runs when the array reference
		// changes (filter/sort swap it for a new array; load-more appends).
		items;
		const teardown = installFeedMasonry();
		void tick().then(() => {
			requestAnimationFrame(() => {
				applyFeedMasonry();
			});
		});
		return teardown;
	});

	function buildURL(nextType: string, nextSort: string, cursor = ""): string {
		const params = new URLSearchParams();
		if (nextType) params.set("type", nextType);
		if (nextSort && nextSort !== "recent") params.set("sort", nextSort);
		if (cursor) params.set("cursor", cursor);
		const query = params.toString();
		return `/api/feed${query ? `?${query}` : ""}`;
	}

	// Update the URL query string (shallow) when filters change so the
	// state is shareable and survives back-button.
	function updateURL(nextType: string, nextSort: string) {
		const params = new URLSearchParams();
		if (nextType) params.set("type", nextType);
		if (nextSort && nextSort !== "recent") params.set("sort", nextSort);
		const query = params.toString();
		goto(query ? `/?${query}` : "/", { replaceState: true, keepFocus: true, noScroll: true });
	}

	async function applyFilter(nextType: string) {
		if (typeFilter === nextType && !loading) return;
		typeFilter = nextType;
		loading = true;
		updateURL(nextType, sort);
		try {
			const res = await fetch(buildURL(nextType, sort), {
				headers: { Accept: "application/json" },
			});
			if (!res.ok) throw new Error(`Feed failed: ${res.status}`);
			const data = (await res.json()) as FeedResponse;
			items = data.items ?? [];
			nextCursor = data.next_cursor ?? "";
		} catch {
			pushToast("Failed to load feed");
		} finally {
			loading = false;
		}
	}

	async function applySort(nextSort: string) {
		if ((sort || "recent") === nextSort && !loading) return;
		sort = nextSort;
		loading = true;
		updateURL(typeFilter, nextSort);
		try {
			const res = await fetch(buildURL(typeFilter, nextSort), {
				headers: { Accept: "application/json" },
			});
			if (!res.ok) throw new Error(`Feed failed: ${res.status}`);
			const data = (await res.json()) as FeedResponse;
			items = data.items ?? [];
			nextCursor = data.next_cursor ?? "";
		} catch {
			pushToast("Failed to load feed");
		} finally {
			loading = false;
		}
	}

	async function loadMore() {
		if (!nextCursor || loadingMore) return;
		loadingMore = true;
		try {
			const res = await fetch(buildURL(typeFilter, sort, nextCursor), {
				headers: { Accept: "application/json" },
			});
			if (!res.ok) throw new Error(`Load more failed: ${res.status}`);
			const data = (await res.json()) as FeedResponse;
			items = [...items, ...(data.items ?? [])];
			nextCursor = data.next_cursor ?? "";
		} catch {
			pushToast("Failed to load more");
		} finally {
			loadingMore = false;
		}
	}

	let isAuthenticated = $derived(data.isAuthenticated);
	let appName = $derived(data.appName ?? "arabica");
	let showFilters = $derived(true); // Filters are useful for all viewers.
</script>

<svelte:head>
	<title>Arabica — Coffee Brew Tracking on AT Protocol</title>
	<meta name="description" content="Log every brew, track your beans and equipment, and share your coffee story with the community. Built on AT Protocol — you own your data." />
</svelte:head>

<div class="page-container-lg">
	{#if appName === "oolong"}
		<div class="alert-warning mb-4">
			<strong>Heads up:</strong> oolong is <em>very</em> new and unstable (initial release was May 17th). Breaking lexicon changes <strong>will</strong> happen frequently.
		</div>
	{/if}

	{#if isAuthenticated}
		<!-- Welcome card (authenticated) -->
		<div class="text-center mb-8 pt-4">
			<h1 class="text-3xl sm:text-4xl font-semibold text-primary mb-6">
				{appName === "oolong" ? "Your tea journey, documented." : "Your coffee journey, documented."}
			</h1>
			<div class="grid grid-cols-2 sm:grid-cols-4 gap-3 max-w-xl mx-auto">
				<a href="/brews/new" class="home-action-primary block text-center py-4 px-4 rounded-xl">
					<span class="text-base font-semibold">{appName === "oolong" ? "Log Steep" : "Log Brew"}</span>
				</a>
				<a href="/explore" class="home-action-secondary block text-center py-4 px-4 rounded-xl">
					<span class="text-base font-semibold">Explore</span>
				</a>
				<a href="/my-coffee" class="home-action-secondary block text-center py-4 px-4 rounded-xl">
					<span class="text-base font-semibold">{appName === "oolong" ? "My Tea" : "My Coffee"}</span>
				</a>
				<a href="/profile/{data.userDID}" class="home-action-secondary block text-center py-4 px-4 rounded-xl">
					<span class="text-base font-semibold">Profile</span>
				</a>
			</div>
		</div>
	{:else}
		<!-- Welcome hero (unauthenticated) -->
		<div class="text-center mb-8 pt-4">
			<h1 class="text-3xl sm:text-4xl font-semibold text-primary mb-3">
				{appName === "oolong" ? "Your tea journey, documented." : "Your coffee journey, documented."}
			</h1>
			<p class="text-lg text-emphasis max-w-xl mx-auto mb-3">
				{appName === "oolong"
					? "Log every steep, track your teaware and vendors, and share your tea story with the community."
					: "Log every brew, track your beans and equipment, and share your coffee story with the community."}
			</p>
			<p class="text-sm text-faint">
				<a href="/atproto" class="hover:text-emphasis transition-colors">Built on AT Protocol</a> — you own your data.
			</p>
		</div>
		<!-- Login card -->
		<div class="card p-6 mb-8 max-w-md mx-auto">
			<h2 class="text-lg font-semibold text-primary mb-4 text-center">Log in to get started</h2>
			<form method="POST" action="/auth/login">
				<input
					type="text"
					id="handle"
					name="handle"
					placeholder="your-handle.bsky.social"
					autocomplete="off"
					required
					class="w-full form-input-lg"
				/>
				<button type="submit" class="btn-primary w-full mt-3 py-3 px-8 font-semibold">Log In</button>
			</form>
			<div class="mt-4 text-sm text-muted text-center">
				<a href="/join/create" class="font-medium text-secondary hover:text-primary transition-colors hover:underline">Create an account</a>
				<span class="mx-1.5 text-placeholder">·</span>
				<a href="/about" class="text-muted hover:text-secondary transition-colors hover:underline">Learn more</a>
			</div>
		</div>
	{/if}

	<!-- Community feed -->
	<div class="card p-2 sm:p-6 mb-8">
		<h3 class="text-xl font-bold text-primary mb-4">Community Activity</h3>
		{#if showFilters}
			<FeedFilters
				{typeFilter}
				{sort}
				{loading}
				onType={applyFilter}
				onSort={applySort}
			/>
		{/if}
		<div id="feed-board" class="feed-board">
			<div id="feed-items" class="feed-grid" data-feed-masonry>
				{#if data.error && items.length === 0}
					<div class="rounded-xl p-8 text-center" style="grid-column: 1 / -1;">
						<div class="text-4xl mb-3">✦</div>
						<p class="text-emphasis font-medium mb-1">The feed is quiet today</p>
						<p class="text-sm text-faint">{data.error}</p>
					</div>
				{:else if loading && items.length === 0}
					<!-- Loading skeleton -->
					<div class="space-y-4 animate-pulse" style="grid-column: 1 / -1;">
						<div class="section-box">
							<div class="flex items-center gap-3 mb-3">
								<div class="w-10 h-10 rounded-full bg-brown-300"></div>
								<div class="flex-1">
									<div class="h-4 bg-brown-300 rounded-sm w-1/4 mb-2"></div>
									<div class="h-3 bg-brown-200 rounded-sm w-1/6"></div>
								</div>
							</div>
							<div class="bg-brown-200 rounded-lg p-3">
								<div class="h-4 bg-brown-300 rounded-sm w-3/4 mb-2"></div>
								<div class="h-3 bg-brown-200 rounded-sm w-1/2"></div>
							</div>
						</div>
					</div>
				{:else if items.length === 0}
					<div class="rounded-xl p-8 text-center" style="grid-column: 1 / -1; background: var(--surface-bg); border: 2px dashed var(--card-border);">
						<div class="text-4xl mb-3">✦</div>
						<p class="text-emphasis font-medium mb-1">The feed is quiet today</p>
						<p class="text-sm text-faint">Follow people or add your first record to get started.</p>
					</div>
				{:else}
					{#each items as item (item.subject_uri)}
						<FeedCard {item} {isAuthenticated} />
					{/each}
					{#if nextCursor}
						<div class="text-center pt-2" style="grid-column: 1 / -1;">
							<button
								class="btn-secondary text-sm load-more-btn"
								disabled={loadingMore}
								onclick={loadMore}
							>
								{loadingMore ? "Loading..." : "Load more"}
							</button>
						</div>
					{/if}
				{/if}
			</div>
		</div>
	</div>

	{#if isAuthenticated}
		<!-- About info card -->
		<div class="card p-6 mb-8">
			<h3 class="text-lg font-bold text-primary mb-2">About Arabica</h3>
			<p class="text-sm text-secondary">
				Arabica is a coffee brew tracking app built on AT Protocol. Your brewing data is stored in your own Personal Data Server, giving you full ownership and portability.
			</p>
			<a href="/about" class="text-sm text-secondary hover:text-primary hover:underline mt-2 inline-block">Learn more →</a>
		</div>
	{/if}
</div>
