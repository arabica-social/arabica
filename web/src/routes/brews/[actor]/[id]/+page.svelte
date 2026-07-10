<script lang="ts">
	import BackButton from "$lib/components/BackButton.svelte";
	import RecordViewHeader from "$lib/components/RecordViewHeader.svelte";
	import ActionBar from "$lib/components/ActionBar.svelte";
	import CommentSection from "$lib/components/CommentSection.svelte";
	import Icon from "$lib/components/Icon.svelte";
	import { formatTime, formatTemp } from "$lib/utils/format";
	import { pushToast } from "$lib/stores/toasts";
	import { session } from "$lib/stores/session";
	import type { PageData } from "./$types";
	import type { Brew, Pour } from "$lib/types/entity_view";

	let { data }: { data: PageData } = $props();

	let b = $derived(data.data?.record as Brew | undefined);
	let v = $derived(data.data);

	function ownerFromShareURL(shareURL: string): string {
		const parts = shareURL.replace(/^\//, "").split("/");
		return parts.length >= 2 ? parts[1] : "";
	}

	function brewTitle(brew: Brew): string {
		if (brew.bean) return brew.bean.name || brew.bean.origin;
		return "Coffee Brew";
	}

	function totalWater(brew: Brew): number {
		if (brew.water_amount > 0) return brew.water_amount;
		return (brew.pours ?? []).reduce((sum, p) => sum + (p?.water_amount ?? 0), 0);
	}

	function ratioDisplay(brew: Brew): string {
		const water = totalWater(brew);
		if (brew.coffee_amount > 0 && water > 0) {
			return `1:${(water / brew.coffee_amount).toFixed(1)}`;
		}
		return "";
	}

	function timeDisplay(brew: Brew): string {
		if (brew.time_seconds > 0) return formatTime(brew.time_seconds);
		return "";
	}

	function brewerName(brew: Brew): string {
		return brew.brewer_obj?.name ?? brew.method ?? "";
	}

	function grinderName(brew: Brew): string {
		return brew.grinder_obj?.name ?? "";
	}

	function grinderViewURL(brew: Brew, owner: string): string {
		if (brew.grinder_obj?.rkey && owner) return `/grinders/${owner}/${brew.grinder_obj.rkey}`;
		return "";
	}

	function brewerViewURL(brew: Brew, owner: string): string {
		if (brew.brewer_obj?.rkey && owner) return `/brewers/${owner}/${brew.brewer_obj.rkey}`;
		return "";
	}

	function recipeViewURL(brew: Brew, owner: string): string {
		if (!brew.recipe_obj?.rkey) return "";
		const o = brew.recipe_obj.author_did || owner;
		return o ? `/recipes/${o}/${brew.recipe_obj.rkey}` : "";
	}

	function formatBloom(bloomWater: number, bloomSeconds: number): string {
		if (bloomWater > 0 && bloomSeconds > 0) return `${bloomWater}g for ${bloomSeconds}s`;
		if (bloomWater > 0) return `${bloomWater}g`;
		if (bloomSeconds > 0) return `${bloomSeconds}s`;
		return "";
	}

	let owner = $derived(v ? ownerFromShareURL(v.share_url) : "");
	let timestampISO = $derived(b ? new Date(b.created_at).toISOString() : "");
	let timestampDisplay = $derived(
		b ? new Date(b.created_at).toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric", hour: "numeric", minute: "2-digit" }) : "",
	);

	async function saveAsRecipe() {
		if (!b) return;
		// The from-brew handler requires a recipe name. Prompt for one,
		// defaulting to the brew's bean name when available.
		const defaultName = b.bean ? (b.bean.name || b.bean.origin) : "";
		const name = window.prompt("Recipe name", defaultName);
		if (!name || !name.trim()) return;
		try {
			const res = await fetch(`/api/recipes/from-brew/${b.rkey}`, {
				method: "POST",
				headers: { "Content-Type": "application/x-www-form-urlencoded" },
				body: new URLSearchParams({ name: name.trim() }),
				credentials: "same-origin",
			});
			if (!res.ok) throw new Error(`Failed: ${res.status}`);
			const recipe = await res.json();
			pushToast("Recipe saved!");
			// The new recipe lives in the current user's PDS.
			const me = $session.did || $session.handle;
			if (recipe.rkey && me) window.location.href = `/recipes/${encodeURIComponent(me)}/${recipe.rkey}`;
		} catch (error) {
			console.error("Save as recipe failed:", error);
			pushToast("Failed to save recipe");
		}
	}
</script>

<svelte:head>
	<title>{b ? brewTitle(b) : "Brew"} - Arabica</title>
	{#if b}
		<meta name="description" content={brewTitle(b)} />
		<meta property="og:title" content={brewTitle(b)} />
		<meta property="og:type" content="article" />
	{/if}
</svelte:head>

{#if data.error}
	<div class="page-container-sm">
		<div class="card card-inner text-center py-8">
			<p class="text-secondary text-lg mb-2">{data.error}</p>
			<a href="/" class="btn-primary">Back to Home</a>
		</div>
	</div>
{:else if v && b}
	<div class="page-container-sm">
		<div class="card card-inner">
			<RecordViewHeader
				recordType="brew"
				title={brewTitle(b)}
				timestamp={timestampDisplay}
				{timestampISO}
				authorDID={v.author?.did}
				authorHandle={v.author?.handle}
				authorDisplay={v.author?.display_name}
				authorAvatar={v.author?.avatar}
			/>
			<div class="record-journal p-4">
				<!-- Brew summary: rating + stats -->
				{#if b.rating > 0 || ratioDisplay(b) || timeDisplay(b) || brewerName(b)}
					<div class="brew-summary">
						{#if b.rating > 0}
							<div class="brew-rating-hero">
								<span class="brew-rating-value">{b.rating}</span>
								<span class="brew-rating-max">/10</span>
							</div>
						{/if}
						{#if ratioDisplay(b) || timeDisplay(b) || brewerName(b)}
							<dl class="brew-summary-stats">
								{#if ratioDisplay(b)}
									<div class="brew-summary-stat"><dt>Ratio</dt><dd>{ratioDisplay(b)}</dd></div>
								{/if}
								{#if timeDisplay(b)}
									<div class="brew-summary-stat"><dt>Time</dt><dd>{timeDisplay(b)}</dd></div>
								{/if}
								{#if brewerName(b)}
									<div class="brew-summary-stat"><dt>Method</dt><dd>{brewerName(b)}</dd></div>
								{/if}
							</dl>
						{/if}
					</div>
				{/if}

				<!-- Bean reference card -->
				{#if b.bean}
					<div class="journal-bean-ref mb-2">
						<span class="detail-label mb-2 block">
							<span class="inline-flex items-center gap-1">
								<Icon name="bean" class="w-4 h-4 text-muted" />
								Coffee Bean
							</span>
						</span>
						<a href={`/beans/${owner}/${b.bean.rkey}`} class="detail-value-lg hover:underline">
							{b.bean.name || b.bean.origin}
						</a>
						{#if b.bean.roaster?.name}
							<div class="text-sm mt-1 text-secondary">
								<span class="inline-flex items-center gap-1">
									<Icon name="store" class="w-4 h-4 text-muted" />
									<a href={`/roasters/${owner}/${b.bean.roaster.rkey}`} class="hover:underline">
										{b.bean.roaster.name}
									</a>
								</span>
							</div>
						{/if}
						<div class="flex flex-wrap gap-3 mt-2 text-sm text-muted">
							{#if b.bean.origin}
								<span class="inline-flex items-center gap-1">
									<Icon name="mapPin" class="w-4 h-4 text-red-400" />
									{b.bean.origin}
								</span>
							{/if}
							{#if b.bean.roast_level}
								<span class="inline-flex items-center gap-1">
									<Icon name="flame" class="w-4 h-4 text-orange-400" />
									{b.bean.roast_level}
								</span>
							{/if}
						</div>
					</div>
				{/if}

				<!-- Inputs -->
				<div class="my-6">
					<div class="ledger-section">Inputs</div>
					{#if b.coffee_amount > 0}
						<div class="journal-field">
							<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="scale" class="w-4 h-4 text-faint" />Coffee</span></span>
							<span class="detail-value">{b.coffee_amount}g</span>
						</div>
					{/if}
					{#if totalWater(b) > 0}
						<div class="journal-field">
							<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="droplet" class="w-4 h-4 text-blue-400" />Water</span></span>
							<span class="detail-value">{totalWater(b)}g</span>
						</div>
					{/if}
					{#if grinderName(b)}
						<div class="journal-field">
							<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="gear" class="w-4 h-4 text-faint" />Grinder</span></span>
							{#if grinderViewURL(b, owner)}
								<a href={grinderViewURL(b, owner)} class="detail-value hover:underline">{grinderName(b)}</a>
							{:else}
								<span class="detail-value">{grinderName(b)}</span>
							{/if}
						</div>
					{/if}
					{#if b.grind_size}
						<div class="journal-field">
							<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="disc" class="w-4 h-4 text-placeholder" />Grind Size</span></span>
							<span class="detail-value">{b.grind_size}{#if grinderName(b)} ({grinderName(b)}){/if}</span>
						</div>
					{/if}
					{#if b.temperature > 0}
						<div class="journal-field">
							<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="thermometer" class="w-4 h-4 text-red-400" />Temperature</span></span>
							<span class="detail-value">{formatTemp(b.temperature)}</span>
						</div>
					{/if}
					{#if b.pourover_params?.filter}
						<div class="journal-field">
							<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="sliders" class="w-4 h-4 text-placeholder" />Filter</span></span>
							<span class="detail-value">{b.pourover_params.filter}</span>
						</div>
					{/if}
				</div>

				<!-- Process -->
				<div>
					<div class="ledger-section">Process</div>
					{#if brewerName(b)}
						<div class="journal-field">
							<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="coffee" class="w-4 h-4 text-muted" />Brew Method</span></span>
							{#if brewerViewURL(b, owner)}
								<a href={brewerViewURL(b, owner)} class="detail-value hover:underline">{brewerName(b)}</a>
							{:else}
								<span class="detail-value">{brewerName(b)}</span>
							{/if}
						</div>
					{/if}
					{#if b.time_seconds > 0}
						<div class="journal-field">
							<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="clock" class="w-4 h-4 text-placeholder" />Brew Time</span></span>
							<span class="detail-value">{timeDisplay(b)}</span>
						</div>
					{/if}
					{#if b.espresso_params}
						{#if b.espresso_params.pre_infusion_seconds > 0}
							<div class="journal-field">
								<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="clock" class="w-4 h-4 text-placeholder" />Pre-infusion</span></span>
								<span class="detail-value">{b.espresso_params.pre_infusion_seconds}s</span>
							</div>
						{/if}
						{#if b.espresso_params.pressure > 0}
							<div class="journal-field">
								<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="sliders" class="w-4 h-4 text-placeholder" />Pressure</span></span>
								<span class="detail-value">{b.espresso_params.pressure.toFixed(1)} bar</span>
							</div>
						{/if}
					{/if}
					{#if b.pourover_params}
						{#if b.pourover_params.bloom_water > 0 || b.pourover_params.bloom_seconds > 0}
							<div class="journal-field">
								<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="droplet" class="w-4 h-4 text-blue-400" />Bloom</span></span>
								<span class="detail-value">{formatBloom(b.pourover_params.bloom_water, b.pourover_params.bloom_seconds)}</span>
							</div>
						{/if}
						{#if b.pourover_params.drawdown_seconds > 0}
							<div class="journal-field">
								<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="clock" class="w-4 h-4 text-placeholder" />Drawdown</span></span>
								<span class="detail-value">{b.pourover_params.drawdown_seconds}s</span>
							</div>
						{/if}
						{#if b.pourover_params.bypass_water > 0}
							<div class="journal-field">
								<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="droplet" class="w-4 h-4 text-blue-400" />Bypass Water</span></span>
								<span class="detail-value">{b.pourover_params.bypass_water}g</span>
							</div>
						{/if}
					{/if}
					{#if b.espresso_params && b.espresso_params.yield_weight > 0}
						<div class="ledger-section">Output</div>
						<div class="journal-field">
							<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="scale" class="w-4 h-4 text-faint" />Yield</span></span>
							<span class="detail-value">{b.espresso_params.yield_weight.toFixed(1)}g</span>
						</div>
					{/if}
				</div>

				<!-- Recipe / Pours / Tasting notes -->
				<div class="space-y-6">
					{#if b.recipe_obj}
						<div>
							<span class="detail-label mb-2 block">Recipe</span>
							{#if recipeViewURL(b, owner)}
								<a href={recipeViewURL(b, owner)} class="detail-value-lg hover:underline">{b.recipe_obj.name}</a>
							{:else}
								<span class="detail-value-lg">{b.recipe_obj.name}</span>
							{/if}
						</div>
					{/if}
					{#if b.pours && b.pours.length > 0}
						<div>
							<span class="detail-label mb-3 block">
								<span class="inline-flex items-center gap-1"><Icon name="droplet" class="w-4 h-4 text-blue-400" />Pours</span>
							</span>
							<div class="space-y-2">
								{#each b.pours as pour (pour.pour_number)}
									<div class="pour-row">
										<span class="detail-value">{pour.water_amount}g</span>
										<span class="text-muted">for {formatTime(pour.time_seconds)}</span>
									</div>
								{/each}
							</div>
						</div>
					{/if}
					{#if b.tasting_notes}
						<div>
							<span class="detail-label mb-2 block">
								<span class="inline-flex items-center gap-1"><Icon name="fileText" class="w-4 h-4 text-placeholder" />Tasting Notes</span>
							</span>
							<div class="journal-prose">{b.tasting_notes}</div>
						</div>
					{/if}
					{#if v.is_own_profile && !b.recipe_obj}
						<button type="button" class="w-full btn-secondary text-sm" onclick={saveAsRecipe}>
							Save as Recipe
						</button>
					{/if}
				</div>
			</div>

			<div class="record-view-footer">
				<BackButton />
				<ActionBar
					subjectURI={v.subject_uri}
					subjectCID={v.subject_cid}
					isLiked={v.social.is_liked}
					likeCount={v.social.like_count}
					commentCount={v.social.comment_count}
					shareURL={v.share_url}
					shareTitle={brewTitle(b)}
					shareText="Check out this brew on Arabica"
					isOwner={v.is_own_profile}
					editURL={`/brews/${b.rkey}/edit`}
					deleteURL={`/brews/${b.rkey}`}
					deleteRedirect="/my-coffee"
					isAuthenticated={v.is_authenticated}
					isModerator={v.social.is_moderator}
					canHideRecord={v.social.can_hide_record}
					canBlockUser={v.social.can_block_user}
					isRecordHidden={v.social.is_record_hidden}
					authorDID={v.author?.did ?? ""}
				/>
			</div>

			<CommentSection
				subjectURI={v.subject_uri}
				subjectCID={v.subject_cid}
				comments={v.social.comments}
				isAuthenticated={v.is_authenticated}
				currentUserDID=""
				isModerator={v.social.is_moderator}
				viewURL={v.share_url}
			/>
		</div>
	</div>
{/if}
