<script lang="ts">
	import EntityViewLayout from "$lib/components/EntityViewLayout.svelte";
	import Icon from "$lib/components/Icon.svelte";
	import { safeWebsiteURL, pluralS } from "$lib/utils/format";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();

	// Narrow the record to the roaster shape.
	let r = $derived(
		data.data?.record as
			| { name: string; location: string; website: string; rkey: string; created_at: string }
			| undefined,
	);
	let v = $derived(data.data);
	let safeLink = $derived(r ? safeWebsiteURL(r.website) : "");
</script>

<svelte:head>
	<title>{r?.name ?? "Roaster"} - Arabica</title>
	{#if r}
		<meta name="description" content={`${r.name}${r.location ? " — " + r.location : ""}`} />
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
	<EntityViewLayout
		recordType="roaster"
		title={r.name}
		createdAt={r.created_at}
		authorDID={v.author?.did}
		authorHandle={v.author?.handle}
		authorDisplay={v.author?.display_name}
		authorAvatar={v.author?.avatar}
		backlinks={v.backlinks}
		subjectURI={v.subject_uri}
		subjectCID={v.subject_cid}
		isLiked={v.social.is_liked}
		likeCount={v.social.like_count}
		commentCount={v.social.comment_count}
		shareURL={v.share_url}
		isOwner={v.is_own_profile}
		editURL={`/roasters/${r.rkey}/edit`}
		deleteURL={`/api/roasters/${r.rkey}`}
		deleteRedirect="/my-coffee"
		isAuthenticated={v.is_authenticated}
		isModerator={v.social.is_moderator}
		canHideRecord={v.social.can_hide_record}
		canBlockUser={v.social.can_block_user}
		isRecordHidden={v.social.is_record_hidden}
		comments={v.social.comments}
		currentUserDID=""
	>
		<div class="record-label p-4">
			<div class="label-detail">
				<span class="detail-label">
					<span class="inline-flex items-center gap-1">
						<Icon name="mapPin" class="w-4 h-4 text-red-400" />
						Location
					</span>
				</span>
				{#if r.location}
					<span class="detail-value-lg">{r.location}</span>
				{:else}
					<span class="text-sm text-faint">—</span>
				{/if}
			</div>
			{#if safeLink}
				<div class="label-detail">
					<span class="detail-label">
						<span class="inline-flex items-center gap-1">
							<Icon name="fileText" class="w-4 h-4 text-blue-400" />
							Website
						</span>
					</span>
					<a href={safeLink} target="_blank" rel="noopener noreferrer" class="detail-value hover:underline">
						{safeLink}
					</a>
				</div>
			{/if}
		</div>

		{#snippet statLine()}
			{#if v.entity_count > 0}
				<div class="record-stat-line">
					<span class="flex items-center gap-1">
						<Icon name="leaf" class="w-4 h-4 text-green-600" />
						{v.entity_count} bean{pluralS(v.entity_count)}
					</span>
				</div>
			{/if}
		{/snippet}
	</EntityViewLayout>
{/if}
