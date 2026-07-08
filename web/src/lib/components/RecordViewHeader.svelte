<script lang="ts">
	import Avatar from "./Avatar.svelte";
	import TypeBadge from "./TypeBadge.svelte";
	import { displayHandle } from "../stores/session";

	let {
		recordType,
		title,
		timestamp,
		timestampISO,
		authorDID = "",
		authorHandle = "",
		authorDisplay = "",
		authorAvatar = "",
	}: {
		recordType: string;
		title: string;
		timestamp: string;
		timestampISO: string;
		authorDID?: string;
		authorHandle?: string;
		authorDisplay?: string;
		authorAvatar?: string;
	} = $props();
</script>

<div class={`record-view-header record-view-header-${recordType}`}>
	{#if authorHandle}
		<div class="record-view-author">
			<a href={`/profile/${authorHandle}`} class="flex-shrink-0">
				<Avatar avatarURL={authorAvatar} displayName={authorDisplay} size="md" />
			</a>
			<div class="record-view-author-info">
				<div class="record-view-author-names">
					{#if authorDisplay}
						<a href={`/profile/${authorHandle}`} class="record-view-author-displayname">
							{authorDisplay}
						</a>
					{/if}
					<a href={`/profile/${authorHandle}`} class="record-view-author-handle">
						@{displayHandle(authorHandle)}
					</a>
				</div>
				<div class="record-view-meta">
					<time datetime={timestampISO} data-local="long">{timestamp}</time>
				</div>
			</div>
			<TypeBadge {recordType} />
		</div>
	{:else}
		<div class="record-view-meta mb-3">
			<TypeBadge {recordType} />
			<time datetime={timestampISO} data-local="long">{timestamp}</time>
		</div>
	{/if}
	{#if title}
		<h2 class="record-view-title">{title}</h2>
	{/if}
</div>
