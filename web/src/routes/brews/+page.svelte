<script lang="ts">
	import Icon from "$lib/components/Icon.svelte";
	import { formatTemp, formatTime } from "$lib/utils/format";
	import { pushToast } from "$lib/stores/toasts";
	import type { PageData } from "./$types";
	import type { BrewListResponse } from "$lib/types/manage";
	import type { Brew } from "$lib/types/entity_view";
	import { openLoginModal, session } from "$lib/stores/session";
	import { get } from "svelte/store";

	let { data }: { data: PageData } = $props();

	let brewsData = $state<BrewListResponse | null>(null);
	let loadingMore = $state(false);
	let deletingRKey = $state<string | null>(null);

	// Sync load function data into local state so load-more can extend it.
	$effect(() => {
		brewsData = data.brews;
	});

	let error = $derived(data.error);

	// The brew list API returns the viewer's own brews but not their DID.
	// Resolve the owner DID from the session store to build view links
	// (/brews/{did}/{rkey}), matching the profile and my-coffee pages.
	let ownerDID = $derived(get(session).did ?? "");

	async function loadMore() {
		if (!brewsData?.has_more) return;
		loadingMore = true;
		try {
			const res = await fetch(`/api/brews?offset=${brewsData.next_offset}&limit=25`, {
				headers: { Accept: "application/json" },
			});
			if (!res.ok) {
				pushToast("Failed to load more");
				return;
			}
			const next = (await res.json()) as BrewListResponse;
			brewsData = {
				brews: [...(brewsData.brews ?? []), ...next.brews],
				has_more: next.has_more,
				next_offset: next.next_offset,
			};
		} catch {
			pushToast("Failed to load more");
		} finally {
			loadingMore = false;
		}
	}

	async function deleteBrew(brew: Brew) {
		deletingRKey = brew.rkey;
		try {
			const res = await fetch(`/api/brews/${brew.rkey}`, {
				method: "DELETE",
				credentials: "same-origin",
			});
			if (!res.ok) {
				pushToast("Failed to delete brew");
				return;
			}
			if (brewsData) {
				brewsData = {
					...brewsData,
					brews: brewsData.brews.filter((b) => b.rkey !== brew.rkey),
				};
			}
			pushToast("Brew deleted");
		} catch {
			pushToast("Failed to delete brew");
		} finally {
			deletingRKey = null;
		}
	}
</script>

<svelte:head>
	<title>Your Brews - Arabica</title>
	<meta name="description" content="Your brew journal — every cup you've logged." />
</svelte:head>

<div class="page-container-xl">
	<div class="flex items-center justify-between gap-3 mb-6">
		<h2 class="text-2xl font-semibold text-primary">Your Brews</h2>
		<a href="/brews/new" class="btn-primary shadow-lg hover:shadow-xl">+ New Brew</a>
	</div>

	{#if error}
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{error}</p>
			{#if error === "Authentication required"}
				<button type="button" onclick={openLoginModal} class="btn-primary">Log In</button>
			{/if}
		</div>
	{:else if !brewsData || brewsData.brews.length === 0}
		<div class="card card-inner text-center py-8">
			<p class="text-secondary text-lg mb-2">Your brew journal is empty.</p>
			<p class="text-sm text-muted mb-4">
				Log your first cup and start building your coffee story.
			</p>
			<a href="/brews/new" class="btn-primary px-6 py-3">Log Your First Brew</a>
		</div>
	{:else}
		<div class="space-y-3">
			{#each brewsData.brews as brew (brew.rkey)}
				<div class="feed-card feed-card-brew">
					<!-- Header: date + actions -->
					<div class="flex items-center justify-between mb-2">
						<div class="text-sm text-muted">
							{new Date(brew.created_at).toLocaleDateString(undefined, {
								month: "short",
								day: "numeric",
								year: "numeric",
							})}
						</div>
						{#if brew.rating > 0}
							<span class="badge-rating flex items-center gap-1">
								<Icon name="star" class="w-3 h-3 text-amber-500" />
								{brew.rating}/10
							</span>
						{/if}
						<div class="flex items-center gap-1">
							{#if ownerDID}
								<a
									href={`/brews/${ownerDID}/${brew.rkey}`}
									class="text-muted hover:text-primary text-sm font-medium px-2.5 py-1.5 rounded-sm hover:bg-brown-200"
								>View</a>
							{/if}
							<a
								href={`/brews/${brew.rkey}/edit`}
								class="text-muted hover:text-primary text-sm font-medium px-2.5 py-1.5 rounded-sm hover:bg-brown-200"
							>Edit</a>
							<button
								type="button"
								onclick={() => deleteBrew(brew)}
								disabled={deletingRKey === brew.rkey}
								class="text-faint hover:text-secondary text-sm font-medium px-2.5 py-1.5 rounded-sm hover:bg-brown-200 disabled:opacity-50"
							>
								{deletingRKey === brew.rkey ? "Deleting…" : "Delete"}
							</button>
						</div>
					</div>

					<a href={ownerDID ? `/brews/${ownerDID}/${brew.rkey}` : "/brews"} class="block hover:opacity-90 transition-opacity">
						{#if brew.bean}
							<div class="font-bold text-primary">
								{brew.bean.name || brew.bean.origin}
							</div>
							{#if brew.bean.roaster?.name}
								<div class="text-sm text-muted flex items-center gap-1">
									<Icon name="store" class="w-3 h-3" />
									{brew.bean.roaster.name}
								</div>
							{/if}
							<div class="text-xs text-faint mt-1 flex flex-wrap gap-x-2 gap-y-1">
								{#if brew.bean.origin}
									<span class="inline-flex items-center gap-1"
										><Icon name="mapPin" class="w-3 h-3" />{brew.bean.origin}</span
									>
								{/if}
								{#if brew.bean.roast_level}
									<span class="inline-flex items-center gap-1"
										><Icon name="flame" class="w-3 h-3" />{brew.bean.roast_level}</span
									>
								{/if}
								{#if brew.coffee_amount > 0}
									<span class="inline-flex items-center gap-1"
										><Icon name="scale" class="w-3 h-3" />{brew.coffee_amount}g</span
									>
								{/if}
							</div>
						{:else}
							<div class="font-bold text-primary">Brew</div>
						{/if}
						{#if brew.brewer_obj?.name || brew.method}
							<div class="mb-2">
								<span class="text-meta">Brewer:</span>
								<span class="text-sm font-semibold text-primary"
									>{brew.brewer_obj?.name ?? brew.method}</span
								>
							</div>
						{/if}
						<div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-emphasis">
							{#if brew.grinder_obj?.name}
								<div>
									<span class="text-label">Grinder:</span> {brew.grinder_obj.name}{#if brew.grind_size}
										({brew.grind_size}){/if}
								</div>
							{:else if brew.grind_size}
								<div><span class="text-label">Grind:</span> {brew.grind_size}</div>
							{/if}
							{#if brew.water_amount > 0}
								<div><span class="text-label">Water:</span> {brew.water_amount}g</div>
							{/if}
							{#if brew.temperature > 0}
								<div><span class="text-label">Temp:</span> {formatTemp(brew.temperature)}</div>
							{/if}
							{#if brew.time_seconds > 0}
								<div><span class="text-label">Time:</span> {formatTime(brew.time_seconds)}</div>
							{/if}
						</div>
						{#if brew.tasting_notes}
							<div class="text-sm text-secondary mt-1 line-clamp-2">
								{brew.tasting_notes}
							</div>
						{/if}
					</a>
				</div>
			{/each}

			{#if brewsData.has_more}
				<div class="text-center py-4">
					<button
						type="button"
						class="btn-secondary px-6 py-2"
						onclick={loadMore}
						disabled={loadingMore}
					>
						{loadingMore ? "Loading..." : "Load More"}
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>
