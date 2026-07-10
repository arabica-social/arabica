<script lang="ts">
	import EntityViewLayout from "$lib/components/EntityViewLayout.svelte";
	import Icon from "$lib/components/Icon.svelte";
	import { safeWebsiteURL, pluralS } from "$lib/utils/format";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();

	let b = $derived(
		data.data?.record as
			| { name: string; brewer_type: string; description: string; link: string; rkey: string; created_at: string }
			| undefined,
	);
	let v = $derived(data.data);
	let safeLink = $derived(b ? safeWebsiteURL(b.link) : "");
</script>

<svelte:head>
	<title>{b?.name ?? "Brewer"} - Arabica</title>
	{#if b}
		<meta name="description" content={b.name} />
		<meta property="og:title" content={b.name} />
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
	<EntityViewLayout
		recordType="brewer"
		title={b.name}
		createdAt={b.created_at}
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
		editURL={`/brewers/${b.rkey}/edit`}
		deleteURL={`/api/brewers/${b.rkey}`}
		deleteRedirect="/my-coffee"
		isAuthenticated={v.is_authenticated}
		isModerator={v.social.is_moderator}
		canHideRecord={v.social.can_hide_record}
		canBlockUser={v.social.can_block_user}
		isRecordHidden={v.social.is_record_hidden}
		comments={v.social.comments}
		currentUserDID=""
	>
		<div class="record-journal p-4">
			<div class="journal-field">
				<span class="detail-label">
					<span class="inline-flex items-center gap-1">
						<Icon name="coffee" class="w-4 h-4 text-muted" />
						Type
					</span>
				</span>
				{#if b.brewer_type}
					<span class="detail-value">{b.brewer_type}</span>
				{:else}
					<span class="text-placeholder">Not specified</span>
				{/if}
			</div>
			{#if safeLink}
				<div class="journal-field">
					<span class="detail-label">
						<span class="inline-flex items-center gap-1">
							<Icon name="fileText" class="w-4 h-4 text-blue-400" />
							Link
						</span>
					</span>
					<a href={safeLink} target="_blank" rel="noopener noreferrer" class="detail-value hover:underline">{safeLink}</a>
				</div>
			{/if}
			{#if b.description}
				<div class="mt-6">
					<span class="detail-label mb-2 block">
						<span class="inline-flex items-center gap-1">
							<Icon name="fileText" class="w-4 h-4 text-placeholder" />
							Description
						</span>
					</span>
					<div class="journal-prose">{b.description}</div>
				</div>
			{/if}
		</div>

		{#snippet statLine()}
			{#if v.entity_count > 0}
				<div class="record-stat-line">
					<span class="flex items-center gap-1">
						<Icon name="coffee" class="w-4 h-4 text-muted" />
						{v.entity_count} brew{pluralS(v.entity_count)}
					</span>
				</div>
			{/if}
		{/snippet}
	</EntityViewLayout>
{/if}
