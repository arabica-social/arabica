<script lang="ts">
	import FeedCard from "$lib/components/FeedCard.svelte";
	import Icon from "$lib/components/Icon.svelte";
	import { goto } from "$app/navigation";
	import { tick } from "svelte";
	import { pushToast } from "$lib/stores/toasts";
	import { displayHandle } from "$lib/stores/session";
	import { applyFeedMasonry, installFeedMasonry } from "$lib/utils/feedMasonry";
	import type { PageData } from "./$types";
	import type { ExploreResponse, ExploreDocument } from "$lib/types/api";
	import type { FeedItem } from "$lib/types/feed";

	let { data }: { data: PageData } = $props();

	// svelte-ignore state_referenced_locally
	let explore = $state<ExploreResponse | null>(data.explore);
	let loading = $state(false);
	let loadingMore = $state(false);

	// Filter form state — initialized from URL.
	let q = $state("");
	let typeFilter = $state("");
	let sort = $state("recent");
	let origin = $state("");
	let roaster = $state("");
	let roastLevel = $state("");
	let process = $state("");
	let variety = $state("");
	let location = $state("");
	let grinderType = $state("");
	let burrType = $state("");
	let brewerType = $state("");
	let minRating = $state("");
	let ratioMin = $state("");
	let ratioMax = $state("");
	let closed = $state("");

	$effect(() => {
		const u = new URL(window.location.href);
		q = u.searchParams.get("q") ?? "";
		typeFilter = u.searchParams.get("type") ?? "";
		sort = u.searchParams.get("sort") ?? "recent";
		origin = u.searchParams.get("origin") ?? "";
		roaster = u.searchParams.get("roaster") ?? "";
		roastLevel = u.searchParams.get("roast_level") ?? "";
		process = u.searchParams.get("process") ?? "";
		variety = u.searchParams.get("variety") ?? "";
		location = u.searchParams.get("location") ?? "";
		grinderType = u.searchParams.get("grinder_type") ?? "";
		burrType = u.searchParams.get("burr_type") ?? "";
		brewerType = u.searchParams.get("brewer_type") ?? "";
		minRating = u.searchParams.get("min_rating") ?? "";
		ratioMin = u.searchParams.get("ratio_min") ?? "";
		ratioMax = u.searchParams.get("ratio_max") ?? "";
		closed = u.searchParams.get("closed") ?? "";
		explore = data.explore;
	});

	function buildURL(cursor = ""): string {
		const params = new URLSearchParams();
		if (q) params.set("q", q);
		if (typeFilter) params.set("type", typeFilter);
		if (sort && sort !== "recent") params.set("sort", sort);
		if (origin) params.set("origin", origin);
		if (roaster) params.set("roaster", roaster);
		if (roastLevel) params.set("roast_level", roastLevel);
		if (process) params.set("process", process);
		if (variety) params.set("variety", variety);
		if (location) params.set("location", location);
		if (grinderType) params.set("grinder_type", grinderType);
		if (burrType) params.set("burr_type", burrType);
		if (brewerType) params.set("brewer_type", brewerType);
		if (minRating) params.set("min_rating", minRating);
		if (ratioMin) params.set("ratio_min", ratioMin);
		if (ratioMax) params.set("ratio_max", ratioMax);
		if (closed) params.set("closed", closed);
		if (cursor) params.set("cursor", cursor);
		const query = params.toString();
		return `/api/explore${query ? `?${query}` : ""}`;
	}

	function updateURL() {
		const params = new URLSearchParams();
		if (q) params.set("q", q);
		if (typeFilter) params.set("type", typeFilter);
		if (sort && sort !== "recent") params.set("sort", sort);
		if (origin) params.set("origin", origin);
		if (roaster) params.set("roaster", roaster);
		if (roastLevel) params.set("roast_level", roastLevel);
		if (process) params.set("process", process);
		if (variety) params.set("variety", variety);
		if (location) params.set("location", location);
		if (grinderType) params.set("grinder_type", grinderType);
		if (burrType) params.set("burr_type", burrType);
		if (brewerType) params.set("brewer_type", brewerType);
		if (minRating) params.set("min_rating", minRating);
		if (ratioMin) params.set("ratio_min", ratioMin);
		if (ratioMax) params.set("ratio_max", ratioMax);
		if (closed) params.set("closed", closed);
		const query = params.toString();
		goto(query ? `/explore?${query}` : "/explore", { replaceState: true, keepFocus: true, noScroll: true });
	}

	async function applyFilters(e: SubmitEvent) {
		e.preventDefault();
		loading = true;
		updateURL();
		try {
			const res = await fetch(buildURL(), { headers: { Accept: "application/json" } });
			if (!res.ok) throw new Error("Failed");
			explore = (await res.json()) as ExploreResponse;
		} catch {
			pushToast("Failed to search");
		} finally {
			loading = false;
		}
	}

	function clearFilters() {
		q = ""; typeFilter = ""; sort = "recent"; origin = ""; roaster = ""; roastLevel = "";
		process = ""; variety = ""; location = ""; grinderType = ""; burrType = "";
		brewerType = ""; minRating = ""; ratioMin = ""; ratioMax = ""; closed = "";
		goto("/explore");
	}

	async function loadMore() {
		if (!explore?.next_cursor || loadingMore) return;
		loadingMore = true;
		try {
			const res = await fetch(buildURL(explore.next_cursor), { headers: { Accept: "application/json" } });
			if (!res.ok) throw new Error("Failed");
			const next = (await res.json()) as ExploreResponse;
			explore = {
				...explore,
				items: [...(explore?.items ?? []), ...next.items],
				next_cursor: next.next_cursor,
			};
		} catch {
			pushToast("Failed to load more");
		} finally {
			loadingMore = false;
		}
	}

	function hasAdvancedFilters(): boolean {
		return !!(origin || roaster || roastLevel || process || variety || location || grinderType || burrType || brewerType || minRating || ratioMin || ratioMax || closed);
	}

	let items = $derived(explore?.items ?? []);
	let health = $derived(explore?.health);
	let isAuthenticated = $derived(data.isAuthenticated);

	// Feed masonry: re-pack cards into two corkboard columns whenever the
	// result set changes (filter/load-more). Mirrors the home feed page.
	$effect(() => {
		items;
		const teardown = installFeedMasonry();
		void tick().then(() => {
			requestAnimationFrame(() => {
				applyFeedMasonry();
			});
		});
		return teardown;
	});

	// Document stats from the explore response — own rating, "used by N".
	function docFor(item: FeedItem): ExploreDocument | undefined {
		return explore?.documents?.[item.subject_uri];
	}
</script>

<svelte:head>
	<title>Explore - Arabica</title>
	<meta name="description" content="Find beans, roasters, gear, and recipes from the Arabica community." />
</svelte:head>

<section class="explore-page">
	<header class="explore-hero">
		<div class="explore-hero-copy">
			<p class="explore-eyebrow">Community library</p>
			<h1 class="explore-title">Explore records.</h1>
			<p class="explore-lede">Find beans, roasters, gear, and recipes.</p>
		</div>
		{#if health?.Dirty}
			<div class="explore-stale-note" role="status" aria-live="polite">
				Explore is catching up. Results may be slightly stale.
			</div>
		{/if}
	</header>

	<div class="space-y-6">
		<!-- Search & filters -->
		<details class="explore-controls" open>
			<summary class="explore-controls-summary cursor-pointer">
				<span class="explore-controls-title">
					<span class="block text-sm font-semibold text-primary">Search and filters</span>
				</span>
			</summary>
			<form onsubmit={applyFilters} class="explore-controls-form space-y-4">
				<div class="explore-primary-filters grid grid-cols-1 gap-3 items-end">
					<label class="block">
						<span class="block text-sm font-medium text-muted mb-1">Search</span>
						<input class="form-input w-full" type="search" bind:value={q} placeholder="Ethiopia, V60, washed" />
					</label>
					<label class="block">
						<span class="block text-sm font-medium text-muted mb-1">Type</span>
						<select class="form-select w-full" bind:value={typeFilter}>
							<option value="">All records</option>
							<option value="bean">Beans</option>
							<option value="roaster">Roasters</option>
							<option value="grinder">Grinders</option>
							<option value="brewer">Brewers</option>
							<option value="recipe">Recipes</option>
						</select>
					</label>
					<label class="block">
						<span class="block text-sm font-medium text-muted mb-1">Sort</span>
						<select class="form-select w-full" bind:value={sort}>
							<option value="recent">Recent</option>
							<option value="popular">Popular</option>
							<option value="rating_high">Rating high</option>
						</select>
					</label>
				</div>

				<details class="explore-advanced-filters" open={hasAdvancedFilters()}>
					<summary class="text-sm font-medium text-emphasis cursor-pointer">Advanced filters</summary>
					<div class="explore-filter-grid grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Origin</span>
							<input class="form-input w-full" type="text" bind:value={origin} placeholder="Ethiopia, Colombia, Kenya…" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Roaster</span>
							<input class="form-input w-full" type="text" bind:value={roaster} placeholder="Onyx, Sey, Heart…" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Roast level</span>
							<input class="form-input w-full" type="text" bind:value={roastLevel} placeholder="Light, medium, dark…" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Process</span>
							<input class="form-input w-full" type="text" bind:value={process} placeholder="Washed, natural, honey…" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Variety</span>
							<input class="form-input w-full" type="text" bind:value={variety} placeholder="Bourbon, Gesha, Caturra…" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Location</span>
							<input class="form-input w-full" type="text" bind:value={location} placeholder="Portland, Tokyo, London…" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Grinder type</span>
							<input class="form-input w-full" type="text" bind:value={grinderType} placeholder="Hand, electric…" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Burr type</span>
							<input class="form-input w-full" type="text" bind:value={burrType} placeholder="Conical, flat…" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Brewer type</span>
							<input class="form-input w-full" type="text" bind:value={brewerType} placeholder="V60, espresso, aeropress…" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Minimum rating</span>
							<input class="form-input w-full" type="text" bind:value={minRating} placeholder="8" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Ratio min</span>
							<input class="form-input w-full" type="text" bind:value={ratioMin} placeholder="15" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Ratio max</span>
							<input class="form-input w-full" type="text" bind:value={ratioMax} placeholder="17" />
						</label>
						<label class="block">
							<span class="block text-sm font-medium text-muted mb-1">Closed beans</span>
							<select class="form-select w-full" bind:value={closed}>
								<option value="">Any</option>
								<option value="true">Closed</option>
								<option value="false">Open</option>
							</select>
						</label>
					</div>
				</details>
				<div class="flex flex-wrap gap-2 items-center">
					<button class="btn-primary" type="submit" disabled={loading}>{loading ? "Searching..." : "Explore"}</button>
					<a class="btn-secondary" href="/explore">Clear</a>
					{#if items.length > 0}
						<span class="text-sm text-muted ml-auto">{items.length} shown</span>
					{/if}
				</div>
			</form>
		</details>

		<!-- Results -->
		{#if data.error}
			<div class="card card-inner text-center py-8">
				{#if !isAuthenticated}
					<p class="text-secondary mb-4">{data.error}</p>
					<a href="/login" class="btn-primary">Log In</a>
				{:else}
					<p class="text-secondary">{data.error}</p>
				{/if}
			</div>
		{:else if items.length === 0}
			<div class="card card-inner text-center py-12">
				<div class="text-4xl mb-3">🔍</div>
				<p class="text-emphasis font-medium mb-1">No matching records yet.</p>
				<p class="text-sm text-muted">Try fewer filters, or come back after the witness cache sees more community records.</p>
			</div>
		{:else}
			<div class="explore-results feed-grid" data-feed-masonry>
				{#each items as item (item.subject_uri)}
					<FeedCard {item} {isAuthenticated}>
						{#snippet footer()}
							{#if docFor(item)}
								<div class="explore-doc-stats text-xs text-faint px-2 flex gap-3">
									{#if docFor(item)?.SourceRefCount}
										<span>Used by {docFor(item)?.SourceRefCount}</span>
									{/if}
									{#if docFor(item)?.OwnRating?.Valid}
										<span>Your rating: {docFor(item)?.OwnRating.Float64}/10</span>
									{/if}
								</div>
							{/if}
						{/snippet}
					</FeedCard>
				{/each}
			</div>
			{#if explore?.next_cursor}
				<div class="text-center py-4">
					<button type="button" class="btn-secondary" onclick={loadMore} disabled={loadingMore}>
						{loadingMore ? "Loading..." : "More results"}
					</button>
				</div>
			{/if}
		{/if}
	</div>
</section>
