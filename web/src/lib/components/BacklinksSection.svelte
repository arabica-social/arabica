<script lang="ts">
	import type { BacklinksResult, BacklinkEntry } from "../types/entity_view";

	let {
		result,
		detailURL = "",
	}: {
		result?: BacklinksResult | null;
		detailURL?: string;
	} = $props();

	type Avatar = { handle: string; avatarURL: string; title: string; initial: string };

	function entriesToAvatars(entries: BacklinkEntry[], max: number): Avatar[] {
		const out: Avatar[] = [];
		const seen = new Set<string>();
		for (const e of entries) {
			if (seen.has(e.DID)) continue;
			seen.add(e.DID);
			let title = e.DisplayName || e.Handle || e.DID;
			const initial = [...title][0] ?? "?";
			out.push({ handle: e.Handle, avatarURL: e.AvatarURL, title, initial });
			if (out.length >= max) break;
		}
		return out;
	}

	function pluralWord(singular: string, plural: string, n: number): string {
		return n === 1 ? singular : plural;
	}

	function formatAvg(avg: number): string {
		return `${avg.toFixed(1)}/10`;
	}

	function usageRatingPill(group: { RatingCount: number; RatingAverage: number }): string {
		if (group.RatingCount === 0) return "";
		return formatAvg(group.RatingAverage) + " avg";
	}

	function profileURL(handle: string): string {
		return handle ? `/profile/${handle}` : "#";
	}

	let isEmpty = $derived(
		!result || (result.LibraryCount === 0 && result.UsageCount === 0),
	);
</script>

{#if result && !isEmpty}
	<section class="backlinks-section" aria-labelledby="backlinks-heading">
		<h2 id="backlinks-heading" class="backlinks-heading">Community</h2>
		<div class="backlinks-grid">
			{#if result.LibraryCount > 0}
				<div class="backlinks-block">
					<div class="backlinks-label">
						In {result.LibraryCount} {pluralWord("library", "libraries", result.LibraryCount)}
					</div>
					{#if entriesToAvatars(result.LibraryEntries, 5).length > 0}
						<div class="backlinks-avatars" aria-label={`In ${result.LibraryCount} ${pluralWord("library", "libraries", result.LibraryCount)}`}>
							{#each entriesToAvatars(result.LibraryEntries, 5) as a (a.handle)}
								<a href={profileURL(a.handle)} class="backlinks-avatar" title={a.title}>
									{#if a.avatarURL}
										<img src={a.avatarURL} alt={a.title} loading="lazy" />
									{:else}
										<span class="backlinks-avatar-fallback">{a.initial}</span>
									{/if}
								</a>
							{/each}
							{#if result.LibraryCount > entriesToAvatars(result.LibraryEntries, 5).length}
								<span class="backlinks-avatar-more">+{result.LibraryCount - entriesToAvatars(result.LibraryEntries, 5).length}</span>
							{/if}
						</div>
					{/if}
					{#if detailURL}
						<a href={detailURL} class="backlinks-see-all">See all →</a>
					{/if}
				</div>
			{/if}
			{#each result.Usage as group (group.Key)}
				{#if group.Count > 0}
					<div class="backlinks-block">
						<div class="backlinks-label">
							Used in {group.Count} {group.Label}
							{#if usageRatingPill(group)}
								<span class="badge-rating">{usageRatingPill(group)}</span>
							{/if}
						</div>
						{#if entriesToAvatars(group.Entries, 5).length > 0}
							<div class="backlinks-avatars" aria-label={`Used in ${group.Count} ${group.Label}`}>
								{#each entriesToAvatars(group.Entries, 5) as a (a.handle)}
									<a href={profileURL(a.handle)} class="backlinks-avatar" title={a.title}>
										{#if a.avatarURL}
											<img src={a.avatarURL} alt={a.title} loading="lazy" />
										{:else}
											<span class="backlinks-avatar-fallback">{a.initial}</span>
										{/if}
									</a>
								{/each}
								{#if group.Count > entriesToAvatars(group.Entries, 5).length}
									<span class="backlinks-avatar-more">+{group.Count - entriesToAvatars(group.Entries, 5).length}</span>
								{/if}
							</div>
						{/if}
						{#if detailURL}
							<a href={detailURL} class="backlinks-see-all">See all →</a>
						{/if}
					</div>
				{/if}
			{/each}
		</div>
	</section>
{/if}
