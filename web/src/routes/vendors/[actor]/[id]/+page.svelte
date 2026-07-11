<script lang="ts">
	import EntityViewLayout from "$lib/components/EntityViewLayout.svelte";
	import VendorRecord from "$lib/oolong/components/VendorRecord.svelte";
	import type { PageData } from "./$types";
	import type { Vendor } from "$lib/types/generated/oolong_entities";

	let { data }: { data: PageData } = $props();
	let view = $derived(data.data);
	let vendor = $derived(view?.record as Vendor | undefined);
</script>

<svelte:head>
	<title>{vendor?.name ?? "Vendor"} - Oolong</title>
	{#if vendor}
		<meta name="description" content={`${vendor.name}${vendor.location ? " — " + vendor.location : ""}`} />
		<meta property="og:title" content={vendor.name} />
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
{:else if view && vendor}
	<EntityViewLayout
		recordType="vendor"
		title={vendor.name}
		createdAt={vendor.created_at}
		authorDID={view.author?.did}
		authorHandle={view.author?.handle}
		authorDisplay={view.author?.display_name}
		authorAvatar={view.author?.avatar}
		backlinks={view.backlinks}
		backlinksDetailURL={view.backlinks_detail_url}
		subjectURI={view.subject_uri}
		subjectCID={view.subject_cid}
		isLiked={view.social.is_liked}
		likeCount={view.social.like_count}
		commentCount={view.social.comment_count}
		shareURL={view.share_url}
		isOwner={view.is_own_profile}
		editURL={`/vendors/${vendor.rkey}/edit`}
		deleteURL={`/api/vendors/${vendor.rkey}`}
		deleteRedirect="/my-tea"
		isAuthenticated={view.is_authenticated}
		isModerator={view.social.is_moderator}
		canHideRecord={view.social.can_hide_record}
		canBlockUser={view.social.can_block_user}
		isRecordHidden={view.social.is_record_hidden}
		comments={view.social.comments}
		currentUserDID=""
	>
		<VendorRecord {vendor} />
	</EntityViewLayout>
{/if}
