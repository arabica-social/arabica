<script lang="ts">
	import EntityViewLayout from "$lib/components/EntityViewLayout.svelte";
	import Icon from "$lib/components/Icon.svelte";
	import { formatTime } from "$lib/utils/format";
	import { pushToast } from "$lib/stores/toasts";
	import { session } from "$lib/stores/session";
	import type { PageData } from "./$types";
	import type { Recipe } from "$lib/types/entity_view";

	let { data }: { data: PageData } = $props();

	let r = $derived(data.data?.record as Recipe | undefined);
	let v = $derived(data.data);

	function ownerFromShareURL(shareURL: string): string {
		const parts = shareURL.replace(/^\//, "").split("/");
		return parts.length >= 2 ? parts[1] : "";
	}

	let owner = $derived(v ? ownerFromShareURL(v.share_url) : "");

	function brewerLink(): string {
		if (!r?.brewer_obj || !r.brewer_rkey) return "";
		return owner ? `/brewers/${owner}/${r.brewer_rkey}` : "";
	}

	async function forkRecipe() {
		if (!r) return;
		try {
			// The fork handler requires ?owner= (the source recipe owner's DID
			// or handle) to fetch the source record from its PDS. Prefer the
			// author DID from the view payload; fall back to the owner segment
			// of the share URL.
			const sourceOwner = r.author_did ?? v?.author?.did ?? owner;
			const res = await fetch(`/api/recipes/fork/${r.rkey}?owner=${encodeURIComponent(sourceOwner)}`, {
				method: "POST",
				credentials: "same-origin",
			});
			if (!res.ok) throw new Error(`Fork failed: ${res.status}`);
			const forked = await res.json();
			pushToast("Recipe copied to your library!");
			// The forked copy lives in the CURRENT user's PDS, so navigate to
			// the viewer's recipe — not the source owner's.
			const me = $session.did || $session.handle;
			if (forked.rkey && me) window.location.href = `/recipes/${encodeURIComponent(me)}/${forked.rkey}`;
		} catch (error) {
			console.error("Fork failed:", error);
			pushToast("Failed to copy recipe");
		}
	}
</script>

<svelte:head>
	<title>{r?.name ?? "Recipe"} - Arabica</title>
	{#if r}
		<meta name="description" content={r.name} />
		<meta property="og:title" content={r.name} />
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
{:else if v && r}
	<div class="alert-warning mb-6">
		<div class="flex items-start gap-3">
			<span class="text-xl leading-none mt-0.5">&#9888;&#65039;</span>
			<div class="text-sm">
				<p class="font-bold text-base mb-1">Recipes are in early alpha</p>
				<p class="mb-2">
					The recipe format may change significantly as we figure out what works best. If that happens, your brews won't break &ndash; brew fields are filled in from the recipe at creation time, so they stand on their own. Only the recipe record itself would need to be recreated.
				</p>
			</div>
		</div>
	</div>
	<EntityViewLayout
		recordType="recipe"
		title={r.name}
		createdAt={r.created_at}
		authorDID={v.author?.did}
		authorHandle={v.author?.handle}
		authorDisplay={v.author?.display_name}
		authorAvatar={v.author?.avatar}
		backlinks={v.backlinks}
		backlinksDetailURL={v.backlinks_detail_url}
		subjectURI={v.subject_uri}
		subjectCID={v.subject_cid}
		isLiked={v.social.is_liked}
		likeCount={v.social.like_count}
		commentCount={v.social.comment_count}
		shareURL={v.share_url}
		isOwner={v.is_own_profile}
		editURL={`/recipes/${r.rkey}/edit`}
		deleteURL={`/api/recipes/${r.rkey}`}
		deleteRedirect="/my-coffee"
		isAuthenticated={v.is_authenticated}
		isModerator={v.social.is_moderator}
		canHideRecord={v.social.can_hide_record}
		canBlockUser={v.social.can_block_user}
		isRecordHidden={v.social.is_record_hidden}
		comments={v.social.comments}
		currentUserDID=""
	>
		{#snippet statLine()}
			{#if v.extras?.source_recipe_url}
				<div class="-mt-3 mb-3">
					<span class="fork-chip">
						↳ forked from
						<a href={String(v.extras.source_recipe_url)}>
							{v.extras.source_recipe_author
								? `${String(v.extras.source_recipe_author)}'s recipe`
								: "original recipe"}
						</a>
					</span>
				</div>
			{/if}
			<div class="flex items-center gap-3 -mt-3 mb-3">
				{#if v.is_authenticated}
					<a href={`/brews/new?recipe=${r.rkey}&recipe_owner=${r.author_did ?? v.author?.did ?? ""}`} class="btn-primary text-sm text-center">Use in Brew</a>
				{/if}
				{#if v.is_authenticated && !v.is_own_profile}
					<button type="button" class="btn-secondary text-sm" onclick={forkRecipe}>Copy Recipe</button>
				{/if}
			</div>
		{/snippet}

		<div class="record-journal p-4">
			<div>
				{#if r.coffee_amount > 0}
					<div class="journal-field">
						<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="coffee" class="w-4 h-4 text-muted" />Coffee</span></span>
						<span class="detail-value">{r.coffee_amount.toFixed(1)}g</span>
					</div>
				{/if}
				{#if r.water_amount > 0}
					<div class="journal-field">
						<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="droplet" class="w-4 h-4 text-blue-400" />Water</span></span>
						<span class="detail-value">{r.water_amount.toFixed(1)}g</span>
					</div>
				{/if}
				{#if r.ratio && r.ratio > 0}
					<div class="journal-field">
						<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="scale" class="w-4 h-4 text-faint" />Ratio</span></span>
						<span class="detail-value">1:{r.ratio.toFixed(1)}</span>
					</div>
				{/if}
				{#if r.brewer_obj}
					<div class="journal-field">
						<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="brewer" class="w-4 h-4 text-faint" />Brewer</span></span>
						{#if brewerLink()}
							<a href={brewerLink()} class="detail-value hover:underline">{r.brewer_obj.name}</a>
						{:else}
							<span class="detail-value">{r.brewer_obj.name}</span>
						{/if}
					</div>
				{:else if r.brewer_type}
					<div class="journal-field">
						<span class="detail-label"><span class="inline-flex items-center gap-1"><Icon name="brewer" class="w-4 h-4 text-faint" />Brewer Type</span></span>
						<span class="detail-value">{r.brewer_type}</span>
					</div>
				{/if}
			</div>
			<div class="space-y-6 mt-6">
				{#if r.pours && r.pours.length > 0}
					<div>
						<span class="detail-label mb-3 block">
							<span class="inline-flex items-center gap-1"><Icon name="droplet" class="w-4 h-4 text-blue-400" />Pours</span>
						</span>
						<div class="space-y-2">
							{#each r.pours as pour (pour.pour_number)}
								<div class="pour-row">
									<span class="detail-value">{pour.water_amount}g</span>
									<span class="text-muted">for {formatTime(pour.time_seconds)}</span>
								</div>
							{/each}
						</div>
					</div>
				{/if}
				{#if r.notes}
					<div>
						<span class="detail-label mb-2 block">
							<span class="inline-flex items-center gap-1"><Icon name="fileText" class="w-4 h-4 text-placeholder" />Notes</span>
						</span>
						<div class="journal-prose">{r.notes}</div>
					</div>
				{/if}
			</div>
		</div>
	</EntityViewLayout>
{/if}
