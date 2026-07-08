<script lang="ts">
	import BackButton from "$lib/components/BackButton.svelte";
	import RecordViewHeader from "$lib/components/RecordViewHeader.svelte";
	import ActionBar from "$lib/components/ActionBar.svelte";
	import CommentSection from "$lib/components/CommentSection.svelte";
	import BacklinksSection from "$lib/components/BacklinksSection.svelte";
	import Icon from "$lib/components/Icon.svelte";
	import { safeWebsiteURL, pluralS, formatRating } from "$lib/utils/format";
	import { pushToast } from "$lib/stores/toasts";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();

	type BeanRecord = {
		name: string;
		origin: string;
		variety: string;
		roast_level: string;
		roast_date?: string;
		process: string;
		description: string;
		notes: string;
		link: string;
		rating?: number;
		closed: boolean;
		rkey: string;
		created_at: string;
		roaster?: { name: string; location: string; rkey: string };
	};

	let b = $derived(data.data?.record as BeanRecord | undefined);
	let v = $derived(data.data);
	let safeLink = $derived(b ? safeWebsiteURL(b.link) : "");

	function beanTitle(bean: BeanRecord): string {
		return bean.name || bean.origin;
	}

	function roastLevelTagClass(level: string): string {
		const map: Record<string, string> = {
			"Ultra-Light": "label-tag-roast-1",
			Light: "label-tag-roast-2",
			"Medium-Light": "label-tag-roast-3",
			Medium: "label-tag-roast-4",
			"Medium-Dark": "label-tag-roast-5",
			Dark: "label-tag-roast-6",
		};
		return map[level] ?? "";
	}

	function formatRoastDate(date: string): string {
		const parsed = new Date(date + "T00:00:00");
		if (isNaN(parsed.getTime())) return date;
		return parsed.toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" });
	}

	function ownerFromShareURL(shareURL: string): string {
		const parts = shareURL.replace(/^\//, "").split("/");
		return parts.length >= 2 ? parts[1] : "";
	}

	async function toggleClosed() {
		if (!b) return;
		try {
			const res = await fetch(`/api/beans/${b.rkey}`, {
				method: "PUT",
				credentials: "same-origin",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ ...b, closed: !b.closed }),
			});
			if (!res.ok) throw new Error(`Update failed: ${res.status}`);
			pushToast(b.closed ? "Bag reopened" : "Bag closed");
			window.location.reload();
		} catch (error) {
			console.error("Toggle closed failed:", error);
			pushToast("Failed to update bag");
		}
	}

	let timestampISO = $derived(b ? new Date(b.created_at).toISOString() : "");
	let timestampDisplay = $derived(
		b
			? new Date(b.created_at).toLocaleDateString(undefined, {
					year: "numeric",
					month: "long",
					day: "numeric",
					hour: "numeric",
					minute: "2-digit",
				})
			: "",
	);
</script>

<svelte:head>
	<title>{b ? beanTitle(b) : "Bean"} - Arabica</title>
	{#if b}
		<meta name="description" content={`${b.name}${b.origin ? " — " + b.origin : ""}`} />
		<meta property="og:title" content={beanTitle(b)} />
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
				recordType="bean"
				title=""
				timestamp={timestampDisplay}
				{timestampISO}
				authorDID={v.author?.did}
				authorHandle={v.author?.handle}
				authorDisplay={v.author?.display_name}
				authorAvatar={v.author?.avatar}
			/>
			<div class="record-label p-4">
				<div class="bean-hero-row">
					<div class="bean-hero-text">
						<h1 class="label-origin-hero">{beanTitle(b)}</h1>
						{#if b.roaster?.name}
							<div class="label-byline">
								<a href={`/roasters/${ownerFromShareURL(v.share_url)}/${b.roaster.rkey}`}>
									{b.roaster.name}
								</a>
								{#if b.roaster.location}
									<span class="text-faint">· {b.roaster.location}</span>
								{/if}
							</div>
						{/if}
						{#if b.roast_date}
							<div class="label-byline text-faint">Roasted {formatRoastDate(b.roast_date)}</div>
						{/if}
					</div>
					{#if b.rating}
						<div class="brew-rating-hero">
							<span class="brew-rating-value">{b.rating}</span>
							<span class="brew-rating-max">/ 10</span>
						</div>
					{/if}
				</div>
				<div class="label-tags">
					{#if b.origin}
						<span class="label-tag label-tag-origin">{b.origin}</span>
					{/if}
					{#if b.variety}
						<span class="label-tag label-tag-variety">{b.variety}</span>
					{/if}
					{#if b.roast_level}
						<span class={`label-tag ${roastLevelTagClass(b.roast_level)}`}>{b.roast_level}</span>
					{/if}
					{#if b.process}
						<span class="label-tag label-tag-process">{b.process}</span>
					{/if}
					{#if b.closed}
						<span class="label-tag label-tag-closed">Closed</span>
					{/if}
				</div>
				{#if b.description}
					<div class="mt-4">
						<div class="form-fieldset-label">Description</div>
						<div class="label-description mt-1">{b.description}</div>
					</div>
				{/if}
				{#if b.notes}
					<div class="mt-4">
						<div class="form-fieldset-label">Personal notes</div>
						<div class="journal-prose mt-1">{b.notes}</div>
					</div>
				{/if}
				{#if safeLink}
					<div class="mt-4 text-sm inline-flex items-center gap-1">
						<Icon name="fileText" class="w-4 h-4 text-blue-400" />
						<a href={safeLink} target="_blank" rel="noopener noreferrer" class="text-secondary hover:underline">{safeLink}</a>
					</div>
				{/if}
			</div>

			{#if v.entity_count > 0}
				<div class="record-stat-line">
					<span class="flex items-center gap-1">
						<Icon name="coffee" class="w-4 h-4 text-muted" />
						{v.entity_count} brew{pluralS(v.entity_count)}
					</span>
				</div>
			{/if}

			<BacklinksSection result={v.backlinks} />

			<div class="record-view-footer">
				<div class="flex items-center gap-3">
					<BackButton />
					{#if v.is_own_profile}
						<div class="flex items-center gap-3">
							{#if !b.closed}
								<button type="button" class="btn-secondary text-sm text-center" onclick={toggleClosed}>
									Close Bag
								</button>
							{:else}
								<button type="button" class="btn-secondary text-sm text-center" onclick={toggleClosed}>
									Reopen Bag
								</button>
							{/if}
							<a href={`/beans/${b.rkey}/edit`} class="btn-secondary text-sm text-center">
								Edit Bean
							</a>
						</div>
					{/if}
				</div>
				<ActionBar
					subjectURI={v.subject_uri}
					subjectCID={v.subject_cid}
					isLiked={v.social.is_liked}
					likeCount={v.social.like_count}
					commentCount={v.social.comment_count}
					shareURL={v.share_url}
					shareTitle={beanTitle(b)}
					shareText="Check out this bean on Arabica"
					isOwner={v.is_own_profile}
					editURL={`/beans/${b.rkey}/edit`}
					deleteURL={`/api/beans/${b.rkey}`}
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
