<script lang="ts">
	import EntityViewLayout from "$lib/components/EntityViewLayout.svelte";
	import Icon from "$lib/components/Icon.svelte";
	import { safeWebsiteURL, pluralS } from "$lib/utils/format";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();

	let g = $derived(
		data.data?.record as
			| { name: string; grinder_type: string; burr_type: string; notes: string; link: string; rkey: string; created_at: string }
			| undefined,
	);
	let v = $derived(data.data);
	let safeLink = $derived(g ? safeWebsiteURL(g.link) : "");
</script>

<svelte:head>
	<title>{g?.name ?? "Grinder"} - Arabica</title>
	{#if g}
		<meta name="description" content={g.name} />
		<meta property="og:title" content={g.name} />
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
{:else if v && g}
	<EntityViewLayout
		recordType="grinder"
		title={g.name}
		createdAt={g.created_at}
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
		editModalURL={`/api/modals/grinder/${g.rkey}`}
		deleteURL={`/api/grinders/${g.rkey}`}
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
			<div>
				<div class="journal-field">
					<span class="detail-label">
						<span class="inline-flex items-center gap-1">
							<Icon name="gear" class="w-4 h-4 text-faint" />
							Type
						</span>
					</span>
					{#if g.grinder_type}
						<span class="detail-value">{g.grinder_type}</span>
					{:else}
						<span class="text-placeholder">Not specified</span>
					{/if}
				</div>
				<div class="journal-field">
					<span class="detail-label">
						<span class="inline-flex items-center gap-1">
							<Icon name="disc" class="w-4 h-4 text-placeholder" />
							Burr Type
						</span>
					</span>
					{#if g.burr_type}
						<span class="detail-value">{g.burr_type}</span>
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
			</div>
			{#if g.notes}
				<div class="mt-6">
					<span class="detail-label mb-2 block">
						<span class="inline-flex items-center gap-1">
							<Icon name="fileText" class="w-4 h-4 text-placeholder" />
							Notes
						</span>
					</span>
					<div class="journal-prose">{g.notes}</div>
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
