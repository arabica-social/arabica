<script lang="ts">
	import type {
		BacklinksResult,
		BacklinkEntry,
		UsageGroup,
	} from "../types/entity_view";

	type Props = {
		result?: BacklinksResult | null;
		entityNoun: string;
		entityName: string;
		backURL?: string;
	};

	let { result, entityNoun, entityName, backURL = "" }: Props = $props();

	// collection NSID tail (e.g. "social.arabica.alpha.bean" -> "bean") ->
	// route path segment (e.g. "beans"). Mirrors Go's collectionTail +
	// entityRoutePaths (domain.EntityRoute.Path). Only arabica collections
	// appear here today; unknown collections render no record link.
	function recordPath(collection: string): string {
		const tail = collection.includes(".")
			? collection.slice(collection.lastIndexOf(".") + 1)
			: collection;
		switch (tail) {
			case "bean":
				return "beans";
			case "roaster":
				return "roasters";
			case "grinder":
				return "grinders";
			case "brewer":
				return "brewers";
			case "recipe":
				return "recipes";
			case "brew":
				return "brews";
			default:
				return "";
		}
	}

	function recordViewURL(e: BacklinkEntry): string {
		const path = recordPath(e.Collection);
		if (!path || !e.RKey) return "";
		const owner = e.Handle || e.DID;
		if (!owner) return "";
		return `/${path}/${owner}/${e.RKey}`;
	}

	function profileURL(handle: string): string {
		return handle ? `/profile/${handle}` : "#";
	}

	function entryDisplayName(e: BacklinkEntry): string {
		if (e.DisplayName) return e.DisplayName;
		if (e.Handle) return e.Handle;
		return e.DID;
	}

	function formatDate(iso: string): string {
		if (!iso) return "";
		const d = new Date(iso);
		if (isNaN(d.getTime())) return iso;
		return d.toLocaleDateString(undefined, {
			month: "short",
			day: "numeric",
			year: "numeric",
		});
	}

	function formatAvg(avg: number): string {
		return `${avg.toFixed(1)}/10`;
	}

	let isEmpty = $derived(
		!result || (result.LibraryCount === 0 && result.UsageCount === 0),
	);

	function usageSectionID(group: UsageGroup): string {
		return `usage-${group.Key}`;
	}
</script>

<div class="page-container-sm">
	<div class="card card-inner backlinks-shell">
		<div class="backlinks-page">
			{#if backURL}
				<a href={backURL} class="text-sm text-secondary hover:underline"
					>← back to {entityNoun}</a
				>
			{/if}
			<h1 class="page-title mt-2">Community around {entityName}</h1>

			{#if isEmpty}
				<p class="text-secondary mt-4">No community backlinks yet.</p>
			{:else}
				{#if result!.RatingCount > 0}
					<div class="card card-inner backlinks-rating-card mt-5">
						<div class="text-xs text-uppercase text-secondary">
							Community brew rating
						</div>
						<div class="mt-2">
							<div class="brew-rating-hero">
								<span class="brew-rating-value"
									>{result!.RatingAverage.toFixed(1)}</span
								>
								<span class="brew-rating-max">/10</span>
							</div>
							<div class="text-sm text-secondary mt-2">
								avg from {result!.RatingCount}
								{result!.RatingCount === 1 ? "rating" : "ratings"}
							</div>
						</div>
					</div>
				{/if}

				{#if result!.LibraryCount > 0}
					<section class="mt-8">
						<h2 class="backlinks-heading">
							In {result!.LibraryCount}
							{result!.LibraryCount === 1
								? "library"
								: "libraries"}
						</h2>
						<ul class="backlinks-entry-list">
							{#each result!.LibraryEntries as e (e.RecordURI)}
								<li class="backlinks-entry">
									{#if e.Title && recordViewURL(e)}
										<a
											href={recordViewURL(e)}
											class="backlinks-entry-title">{e.Title}</a
										>
										<a
											href={profileURL(e.Handle)}
											class="backlinks-entry-user"
										>
											{#if e.AvatarURL}
												<img
													src={e.AvatarURL}
													alt=""
													class="backlinks-entry-avatar"
													loading="lazy"
												/>
											{/if}
											<span class="text-xs text-secondary"
												>by {entryDisplayName(e)}</span
											>
										</a>
									{:else}
										<a
											href={profileURL(e.Handle)}
											class="backlinks-entry-user"
										>
											{#if e.AvatarURL}
												<img
													src={e.AvatarURL}
													alt=""
													class="backlinks-entry-avatar"
													loading="lazy"
												/>
											{/if}
											<span>{entryDisplayName(e)}</span>
										</a>
									{/if}
									<span class="backlinks-entry-meta">
										{#if e.HasRating}
											<span class="badge-rating">{e.Rating}/10</span>
											<span class="text-faint">·</span>
										{/if}
										{formatDate(e.CreatedAt)}
										{#if e.ChainDepth > 1}
											<span class="text-faint"
												>· fork depth {e.ChainDepth}</span
											>
										{/if}
									</span>
								</li>
							{/each}
						</ul>
					</section>
				{/if}

				{#each result!.Usage as group (group.Key)}
					{#if group.Count > 0}
						<section id={usageSectionID(group)} class="mt-8">
							<h2 class="backlinks-heading">
								Used in {group.Count} {group.Label}
							</h2>
							<ul class="backlinks-entry-list">
								{#each group.Entries as e (e.RecordURI)}
									<li class="backlinks-entry">
										{#if e.Title && recordViewURL(e)}
											<a
												href={recordViewURL(e)}
												class="backlinks-entry-title">{e.Title}</a
											>
											<a
												href={profileURL(e.Handle)}
												class="backlinks-entry-user"
											>
												{#if e.AvatarURL}
													<img
														src={e.AvatarURL}
														alt=""
														class="backlinks-entry-avatar"
														loading="lazy"
													/>
												{/if}
												<span class="text-xs text-secondary"
													>by {entryDisplayName(e)}</span
												>
											</a>
										{:else}
											<a
												href={profileURL(e.Handle)}
												class="backlinks-entry-user"
											>
												{#if e.AvatarURL}
													<img
														src={e.AvatarURL}
														alt=""
														class="backlinks-entry-avatar"
														loading="lazy"
													/>
												{/if}
												<span>{entryDisplayName(e)}</span>
											</a>
										{/if}
										<span class="backlinks-entry-meta">
											{#if e.HasRating}
												<span class="badge-rating">{e.Rating}/10</span>
												<span class="text-faint">·</span>
											{/if}
											{formatDate(e.CreatedAt)}
										</span>
									</li>
								{/each}
							</ul>

							{#if group.PerPage > 0 && group.HasNext}
								<div class="backlinks-more-row">
									<a
										href={`?usage=${group.Key}&page=${group.Page + 1}#${usageSectionID(group)}`}
										class="btn-secondary text-sm load-more-btn"
									>
										Load more
									</a>
								</div>
							{/if}
						</section>
					{/if}
				{/each}
			{/if}
		</div>
	</div>
</div>
