<script lang="ts">
	import Avatar from "$lib/components/Avatar.svelte";
	import Icon from "$lib/components/Icon.svelte";
	import ActionBar from "$lib/components/ActionBar.svelte";
	import { displayHandle, safeAvatarURL } from "$lib/stores/session";
	import { pluralS, formatAvgRating } from "$lib/utils/format";
	import { pushToast } from "$lib/stores/toasts";
	import { goto } from "$app/navigation";
	import type { PageData } from "./$types";
	import type { ProfileResponse } from "$lib/types/api";
	import type { Bean, Brew } from "$lib/types/entity_view";

	let { data }: { data: PageData } = $props();

	type Tab = "brews" | "beans" | "equipment";
	let activeTab = $state<Tab>("brews");

	// svelte-ignore state_referenced_locally
	let profile = $state<ProfileResponse | null>(data.profile);

	// svelte-ignore state_referenced_locally
	$effect(() => {
		profile = data.profile;
	});

	function entityURI(did: string, collection: string, rkey: string): string {
		return `at://${did}/${collection}/${rkey}`;
	}

	function beanBrewCount(did: string, rkey: string): number {
		if (!profile) return 0;
		return profile.bean_brew_counts[entityURI(did, "social.arabica.alpha.bean", rkey)] ?? 0;
	}
	function beanAvgRating(did: string, rkey: string): number {
		if (!profile) return 0;
		return profile.bean_avg_brew_ratings[entityURI(did, "social.arabica.alpha.bean", rkey)] ?? 0;
	}
	function roasterBeanCount(did: string, rkey: string): number {
		if (!profile) return 0;
		return profile.roaster_bean_counts[entityURI(did, "social.arabica.alpha.roaster", rkey)] ?? 0;
	}
	function roasterAvgRating(did: string, rkey: string): number {
		if (!profile) return 0;
		return profile.roaster_avg_brew_ratings[entityURI(did, "social.arabica.alpha.roaster", rkey)] ?? 0;
	}
	function grinderBrewCount(did: string, rkey: string): number {
		if (!profile) return 0;
		return profile.grinder_brew_counts[entityURI(did, "social.arabica.alpha.grinder", rkey)] ?? 0;
	}
	function brewerBrewCount(did: string, rkey: string): number {
		if (!profile) return 0;
		return profile.brewer_brew_counts[entityURI(did, "social.arabica.alpha.brewer", rkey)] ?? 0;
	}

	let did = $derived(profile?.did ?? "");
	let isOwnProfile = $derived(profile?.is_own_profile ?? false);
	let isAuthenticated = $derived(data.isAuthenticated);

	function openBeans(): Bean[] {
		return (profile?.beans ?? []).filter((b) => !b.closed);
	}
	function closedBeans(): Bean[] {
		return (profile?.beans ?? []).filter((b) => b.closed);
	}

	let loadingMore = $state(false);
	async function loadMoreBrews() {
		if (!profile?.brews_has_more || loadingMore) return;
		loadingMore = true;
		try {
			const res = await fetch(`/api/profile/${data.actor}?brews_offset=${profile.brews_next_offset}&brews_limit=25`, {
				headers: { Accept: "application/json" },
			});
			if (!res.ok) throw new Error("Failed");
			const next = (await res.json()) as ProfileResponse;
			if (profile) {
				profile = {
					...profile,
					brews: [...profile.brews, ...next.brews],
					brews_has_more: next.brews_has_more,
					brews_next_offset: next.brews_next_offset,
				};
			}
		} catch {
			pushToast("Failed to load more");
		} finally {
			loadingMore = false;
		}
	}
</script>

<svelte:head>
	<title>{profile?.profile.display_name || profile?.profile.handle || "Profile"} - Arabica</title>
	{#if profile}
		<meta name="description" content={`${profile.profile.display_name || profile.profile.handle}'s coffee profile on Arabica`} />
	{/if}
</svelte:head>

{#if data.error && !profile}
	<div class="page-container-lg">
		<div class="card p-8 text-center">
			<div class="text-6xl mb-4 font-bold text-secondary">404</div>
			<h2 class="text-2xl font-bold text-primary mb-4">Page Not Found</h2>
			<p class="text-emphasis mb-6">{data.error}</p>
			<a href="/" class="btn-primary py-3 px-6">Back to Home</a>
		</div>
	</div>
{:else if profile}
	<div class="page-container-lg">
		<!-- Header -->
		<div class="card p-6 mb-6">
			<div class="flex items-center gap-4">
				<Avatar avatarURL={safeAvatarURL(profile.profile.avatar)} displayName={profile.profile.display_name || profile.profile.handle} size="lg" />
				<div>
					{#if profile.profile.display_name}
						<h1 class="text-2xl font-bold text-primary">{profile.profile.display_name}</h1>
					{/if}
					<p class="text-emphasis">@{displayHandle(profile.profile.handle)}</p>
				</div>
				{#if isOwnProfile}
					<div class="ml-auto">
						<a href="/settings" class="btn-secondary text-sm">Settings</a>
					</div>
				{/if}
			</div>
		</div>

		<!-- Stats -->
		<div class="grid grid-cols-2 md:grid-cols-5 gap-4 mb-6">
			<div class="card card-inner text-center">
				<div class="text-2xl font-bold text-primary">{profile.total_brews}</div>
				<div class="text-xs text-muted">Brews</div>
			</div>
			<div class="card card-inner text-center">
				<div class="text-2xl font-bold text-primary">{profile.beans.length}</div>
				<div class="text-xs text-muted">Beans</div>
			</div>
			<div class="card card-inner text-center">
				<div class="text-2xl font-bold text-primary">{profile.roasters.length}</div>
				<div class="text-xs text-muted">Roasters</div>
			</div>
			<div class="card card-inner text-center">
				<div class="text-2xl font-bold text-primary">{profile.grinders.length}</div>
				<div class="text-xs text-muted">Grinders</div>
			</div>
			<div class="card card-inner text-center">
				<div class="text-2xl font-bold text-primary">{profile.brewers.length}</div>
				<div class="text-xs text-muted">Brewers</div>
			</div>
		</div>

		<!-- Tabs -->
		<div class="card shadow-md mb-4">
			<div class="flex border-b border-brown-300">
				<button
					type="button"
					onclick={() => (activeTab = "brews")}
					class={`flex-1 py-3 px-4 text-center font-medium transition-colors ${activeTab === "brews" ? "tab-underline-active" : "tab-underline-inactive"}`}
				>Brews</button>
				<button
					type="button"
					onclick={() => (activeTab = "beans")}
					class={`flex-1 py-3 px-4 text-center font-medium transition-colors ${activeTab === "beans" ? "tab-underline-active" : "tab-underline-inactive"}`}
				>Beans</button>
				<button
					type="button"
					onclick={() => (activeTab = "equipment")}
					class={`flex-1 py-3 px-4 text-center font-medium transition-colors ${activeTab === "equipment" ? "tab-underline-active" : "tab-underline-inactive"}`}
				>Gear</button>
			</div>
		</div>

		<!-- Brews tab -->
		{#if activeTab === "brews"}
			<div class="space-y-4">
				{#if profile.brews.length === 0}
					<div class="card card-inner text-center py-8">
						{#if isOwnProfile}
							<p class="text-secondary mb-4">No brews recorded yet! Start tracking your coffee journey.</p>
							<a href="/brews/new" class="btn-primary">Add Your First Brew</a>
						{:else}
							<p class="text-secondary">No brews yet.</p>
						{/if}
					</div>
				{:else}
					{#each profile.brews as brew (brew.rkey)}
						<a href={`/brews/${did}/${brew.rkey}`} class="feed-card feed-card-brew block hover:shadow-md transition">
							<div class="flex items-center justify-between mb-2">
								<span class="text-sm text-muted">{new Date(brew.created_at).toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" })}</span>
								{#if brew.rating > 0}
									<span class="badge-rating flex items-center gap-1">
										<Icon name="star" class="w-3 h-3 text-amber-500" />
										{brew.rating}/10
									</span>
								{/if}
							</div>
							{#if brew.bean}
								<div class="font-bold text-primary">{brew.bean.name || brew.bean.origin}</div>
								{#if brew.bean.roaster?.name}
									<div class="text-sm text-muted">{brew.bean.roaster.name}</div>
								{/if}
							{/if}
							{#if brew.tasting_notes}
								<div class="text-sm text-secondary mt-1 line-clamp-2">{brew.tasting_notes}</div>
							{/if}
							<div class="text-xs text-faint mt-2 flex gap-3">
								{#if brew.brewer_obj?.name}<span>{brew.brewer_obj.name}</span>{/if}
								{#if brew.coffee_amount > 0}<span>{brew.coffee_amount}g</span>{/if}
							</div>
						</a>
					{/each}
					{#if profile.brews_has_more}
						<div class="text-center py-4">
							<button type="button" class="btn-secondary px-6 py-2" onclick={loadMoreBrews} disabled={loadingMore}>
								{loadingMore ? "Loading..." : "Load More"}
							</button>
						</div>
					{/if}
				{/if}
			</div>
		{/if}

		<!-- Beans tab -->
		{#if activeTab === "beans"}
			<div class="space-y-6">
				<!-- Open bags -->
				<div>
					<h4 class="text-lg font-semibold text-primary mb-3">Open Bags</h4>
					{#if openBeans().length === 0}
						<div class="card card-inner text-center py-6">
							<p class="text-secondary">{isOwnProfile ? "No open bags yet." : "No open bags yet."}</p>
						</div>
					{:else}
						<div class="space-y-3">
							{#each openBeans() as bean (bean.rkey)}
								<div class="feed-card feed-card-bean">
									<a href={`/beans/${did}/${bean.rkey}`} class="font-bold text-primary hover:underline">{bean.name || bean.origin}</a>
									{#if bean.roaster?.name}
										<div class="text-sm text-muted">{bean.roaster.name}</div>
									{/if}
									<div class="text-xs text-faint mt-1 flex flex-wrap gap-x-3">
										{#if bean.origin}<span>{bean.origin}</span>{/if}
										{#if bean.roast_level}<span>{bean.roast_level}</span>{/if}
									</div>
									{#if beanBrewCount(did, bean.rkey) > 0 || beanAvgRating(did, bean.rkey) > 0}
										<div class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60 flex gap-3">
											{#if beanBrewCount(did, bean.rkey) > 0}<span>{beanBrewCount(did, bean.rkey)} brew{pluralS(beanBrewCount(did, bean.rkey))}</span>{/if}
											{#if beanAvgRating(did, bean.rkey) > 0}<span>{formatAvgRating(beanAvgRating(did, bean.rkey))} avg</span>{/if}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>
				<!-- Roasters -->
				<div>
					<h4 class="text-lg font-semibold text-primary mb-3">Roasters</h4>
					{#if profile.roasters.length === 0}
						<div class="card card-inner text-center py-6"><p class="text-secondary">No roasters yet.</p></div>
					{:else}
						<div class="space-y-3">
							{#each profile.roasters as roaster (roaster.rkey)}
								<div class="feed-card">
									<a href={`/roasters/${did}/${roaster.rkey}`} class="font-bold text-primary hover:underline">{roaster.name}</a>
									{#if roaster.location}<div class="text-sm text-muted">{roaster.location}</div>{/if}
									{#if roasterBeanCount(did, roaster.rkey) > 0 || roasterAvgRating(did, roaster.rkey) > 0}
										<div class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60 flex gap-3">
											{#if roasterBeanCount(did, roaster.rkey) > 0}<span>{roasterBeanCount(did, roaster.rkey)} bean{pluralS(roasterBeanCount(did, roaster.rkey))}</span>{/if}
											{#if roasterAvgRating(did, roaster.rkey) > 0}<span>{formatAvgRating(roasterAvgRating(did, roaster.rkey))} avg</span>{/if}
										</div>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>
				<!-- Closed bags -->
				{#if closedBeans().length > 0}
					<div>
						<h4 class="text-lg font-semibold text-primary mb-3">Closed Bags</h4>
						<div class="space-y-3">
							{#each closedBeans() as bean (bean.rkey)}
								<div class="feed-card feed-card-bean opacity-75">
									<a href={`/beans/${did}/${bean.rkey}`} class="font-bold text-primary hover:underline">{bean.name || bean.origin}</a>
									{#if bean.roaster?.name}<div class="text-sm text-muted">{bean.roaster.name}</div>{/if}
								</div>
							{/each}
						</div>
					</div>
				{/if}
			</div>
		{/if}

		<!-- Equipment tab -->
		{#if activeTab === "equipment"}
			<div class="space-y-6">
				<!-- Grinders -->
				<div>
					<h4 class="text-lg font-semibold text-primary mb-3">Grinders</h4>
					{#if profile.grinders.length === 0}
						<div class="card card-inner text-center py-6"><p class="text-secondary">No grinders yet.</p></div>
					{:else}
						<div class="space-y-3">
							{#each profile.grinders as grinder (grinder.rkey)}
								<div class="feed-card">
									<a href={`/grinders/${did}/${grinder.rkey}`} class="font-bold text-primary hover:underline">{grinder.name}</a>
									<div class="text-sm text-muted">{grinder.grinder_type}{#if grinder.burr_type} · {grinder.burr_type}{/if}</div>
									{#if grinderBrewCount(did, grinder.rkey) > 0}
										<div class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60">{grinderBrewCount(did, grinder.rkey)} brew{pluralS(grinderBrewCount(did, grinder.rkey))}</div>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>
				<!-- Brewers -->
				<div>
					<h4 class="text-lg font-semibold text-primary mb-3">Brewers</h4>
					{#if profile.brewers.length === 0}
						<div class="card card-inner text-center py-6"><p class="text-secondary">No brewers yet.</p></div>
					{:else}
						<div class="space-y-3">
							{#each profile.brewers as brewer (brewer.rkey)}
								<div class="feed-card">
									<a href={`/brewers/${did}/${brewer.rkey}`} class="font-bold text-primary hover:underline">{brewer.name}</a>
									<div class="text-sm text-muted">{brewer.brewer_type}</div>
									{#if brewerBrewCount(did, brewer.rkey) > 0}
										<div class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60">{brewerBrewCount(did, brewer.rkey)} brew{pluralS(brewerBrewCount(did, brewer.rkey))}</div>
									{/if}
								</div>
							{/each}
						</div>
					{/if}
				</div>
			</div>
		{/if}
	</div>
{/if}
