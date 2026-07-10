<script lang="ts">
	import type { Snippet } from "svelte";
	import ActionBar from "./ActionBar.svelte";
	import Avatar from "./Avatar.svelte";
	import TypeBadge from "./TypeBadge.svelte";
	import Icon from "./Icon.svelte";
	import { formatTime, formatTemp, safeWebsiteURL } from "../utils/format";
	import { goto } from "$app/navigation";
	import { displayHandle } from "../stores/session";
	import type { FeedItem } from "../types/feed";
	import type { Bean, Brew, Brewer, Grinder, Recipe, Roaster } from "../types/entity_view";

	type Props = {
		item: FeedItem;
		isAuthenticated: boolean;
		footer?: Snippet;
	};

	let { item, isAuthenticated, footer }: Props = $props();

	// Build the share URL for this item's record: /{noun}/{actor}/{rkey}
	function shareURL(i: FeedItem): string {
		const actor = i.author?.handle || i.author?.did || "";
		const rkey = rkeyOf(i);
		if (!actor || !rkey) return `/profile/${i.author?.did ?? ""}`;
		return `/${nounPath(i.record_type)}/${actor}/${rkey}`;
	}

	function nounPath(rt: string): string {
		switch (rt) {
			case "brew": return "brews";
			case "bean": return "beans";
			case "roaster": return "roasters";
			case "grinder": return "grinders";
			case "brewer": return "brewers";
			case "recipe": return "recipes";
			default: return "";
		}
	}

	function rkeyOf(i: FeedItem): string {
		const r = i.record as { rkey?: string };
		return r?.rkey ?? "";
	}

	function cardClass(): string {
		const noun = item.record_type;
		const compact = ["roaster", "grinder", "brewer"].includes(noun) ? " feed-card-compact" : "";
		return `feed-card feed-card-${noun}${compact}`;
	}

	// Per-entity typed records (derived so the template can narrow safely).
	let brew = $derived(item.record_type === "brew" ? (item.record as Brew) : null);
	let bean = $derived(item.record_type === "bean" ? (item.record as Bean) : null);
	let roaster = $derived(item.record_type === "roaster" ? (item.record as Roaster) : null);
	let grinder = $derived(item.record_type === "grinder" ? (item.record as Grinder) : null);
	let brewer = $derived(item.record_type === "brewer" ? (item.record as Brewer) : null);
	let recipe = $derived(item.record_type === "recipe" ? (item.record as Recipe) : null);
	// Espresso/pourover params as standalone deriveds (avoids optional-chain
	// narrowing issues in the template).
	let ep = $derived(brew?.espresso_params ?? null);
	let pp = $derived(brew?.pourover_params ?? null);

	function actionNoun(): string {
		return item.record_type;
	}

	function shareTitle(): string {
		if (brew?.bean) return brew.bean.name || brew.bean.origin || "Brew";
		if (brew) return "Brew";
		if (bean) return bean.name || bean.origin || "Bean";
		if (roaster) return roaster.name || "Roaster";
		if (grinder) return grinder.name || "Grinder";
		if (brewer) return brewer.name || "Brewer";
		if (recipe) return recipe.name || "Recipe";
		return "Arabica";
	}

	let url = $derived(shareURL(item));
	let title = $derived(shareTitle());
	let editURL = $derived(
		item.is_owner
			? `/${nounPath(item.record_type)}/${rkeyOf(item)}/edit`
			: "",
	);
	let deleteURL = $derived(
		item.is_owner
			? item.record_type === "brew"
				? `/${nounPath(item.record_type)}/${rkeyOf(item)}`
				: `/api/${nounPath(item.record_type)}/${rkeyOf(item)}`
			: "",
	);

	function bloomText(pp: NonNullable<Brew["pourover_params"]>): string {
		const parts: string[] = [];
		if (pp.bloom_water > 0) parts.push(`${pp.bloom_water}g`);
		if (pp.bloom_seconds > 0) parts.push(`${pp.bloom_seconds}s`);
		return parts.join(" for ");
	}
</script>

<div class={cardClass()}>
	<!-- Author row -->
	<div class="mb-3">
		<div class="flex items-center gap-2">
			<a href={`/profile/${item.author.did}`}>
				<Avatar avatarURL={item.author.avatar} displayName={item.author.display_name || item.author.handle} size="md" />
			</a>
			<div class="flex-1 min-w-0">
				<a href={`/profile/${item.author.did}`} class="font-medium text-primary hover:underline text-sm">
					{item.author.display_name || displayHandle(item.author.handle)}
				</a>
				<div class="text-xs text-muted">@{displayHandle(item.author.handle)} · {item.time_ago}</div>
			</div>
		</div>
	</div>

	<!-- Action header -->
	<div class="mb-2 text-sm text-emphasis">
		added a
		<a href={url} class="underline hover:text-primary">new {actionNoun()}</a>
		<TypeBadge recordType={actionNoun()} />
	</div>

	<!-- Record content (per entity) — clickable card surface -->
	<div
		role="link"
		tabindex="0"
		class="block hover:opacity-90 transition-opacity cursor-pointer"
		onclick={() => goto(url)}
		onkeydown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); goto(url); } }}
	>
		{#if brew}
			<div class="feed-content-box">
				<div class="flex items-start justify-between gap-3 mb-3">
					<div class="flex-1 min-w-0">
						{#if brew.bean}
							<div class="font-bold text-primary text-base">{brew.bean.name || brew.bean.origin}</div>
							{#if brew.bean.roaster?.name}
								<div class="text-sm text-emphasis mt-0.5 flex items-center gap-1">
									<Icon name="store" class="w-3 h-3" />
									{brew.bean.roaster.name}
								</div>
							{/if}
							<div class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1">
								{#if brew.bean.origin}<span class="flex items-center gap-1"><Icon name="mapPin" class="w-3 h-3" />{brew.bean.origin}</span>{/if}
								{#if brew.bean.roast_level}<span class="flex items-center gap-1"><Icon name="flame" class="w-3 h-3" />{brew.bean.roast_level}</span>{/if}
								{#if brew.coffee_amount > 0}<span class="flex items-center gap-1"><Icon name="scale" class="w-3 h-3" />{brew.coffee_amount}g</span>{/if}
							</div>
						{:else}
							<div class="font-bold text-primary text-base">Brew</div>
						{/if}
					</div>
					{#if brew.rating > 0}
						<span class="badge-rating flex items-center gap-1">
							<Icon name="star" class="w-3 h-3 text-amber-500" />
							{brew.rating}/10
						</span>
					{/if}
				</div>
				{#if brew.brewer_obj?.name || brew.method}
					<div class="mb-2">
						<span class="text-meta">Brewer:</span>
						<span class="text-sm font-semibold text-primary">{brew.brewer_obj?.name ?? brew.method}</span>
					</div>
				{/if}
				<div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-emphasis">
					{#if brew.grinder_obj?.name}
						<div><span class="text-label">Grinder:</span> {brew.grinder_obj.name}{#if brew.grind_size} ({brew.grind_size}){/if}</div>
					{:else if brew.grind_size}
						<div><span class="text-label">Grind:</span> {brew.grind_size}</div>
					{/if}
					{#if brew.water_amount > 0}<div><span class="text-label">Water:</span> {brew.water_amount}g</div>{/if}
					{#if brew.temperature > 0}<div><span class="text-label">Temp:</span> {formatTemp(brew.temperature)}</div>{/if}
					{#if brew.time_seconds > 0}<div><span class="text-label">Time:</span> {formatTime(brew.time_seconds)}</div>{/if}
					{#if ep && ep.yield_weight > 0}<div><span class="text-label">Yield:</span> {ep.yield_weight.toFixed(1)}g</div>{/if}
					{#if ep && ep.pressure > 0}<div><span class="text-label">Pressure:</span> {ep.pressure.toFixed(1)} bar</div>{/if}
					{#if ep && ep.pre_infusion_seconds > 0}<div><span class="text-label">Pre-infusion:</span> {ep.pre_infusion_seconds}s</div>{/if}
					{#if pp && (pp.bloom_water > 0 || pp.bloom_seconds > 0)}
						<div><span class="text-label">Bloom:</span> {bloomText(pp)}</div>
					{/if}
					{#if pp && pp.drawdown_seconds > 0}<div><span class="text-label">Drawdown:</span> {pp.drawdown_seconds}s</div>{/if}
					{#if pp && pp.bypass_water > 0}<div><span class="text-label">Bypass:</span> {pp.bypass_water}g</div>{/if}
					{#if pp && pp.filter}<div><span class="text-label">Filter:</span> {pp.filter}</div>{/if}
				</div>
				{#if brew.pours && brew.pours.length > 0}
					<div class="mt-2 flex flex-wrap items-center gap-1.5">
						<span class="text-xs text-label">Pours:</span>
						{#each brew.pours as pour, i (pour.pour_number)}
							<span class="inline-flex items-center gap-1.5 bg-brown-50 rounded-md px-2 py-1 border border-brown-200 text-xs">
								<span class="font-medium text-emphasis">{i + 1}</span>
								<span class="text-primary">{pour.water_amount}g</span>
								{#if pour.time_seconds > 0}
									<span class="text-placeholder">·</span>
									<span class="text-faint">{formatTime(pour.time_seconds)}</span>
								{/if}
							</span>
						{/each}
					</div>
				{/if}
				{#if brew.tasting_notes}
					<div class="mt-3 text-sm text-secondary italic border-t border-brown-200 pt-2">"{brew.tasting_notes}"</div>
				{/if}
			</div>
		{:else if bean}
			<div class="feed-content-box-sm">
				<div class="font-bold text-primary text-base">{bean.name || bean.origin}</div>
				{#if bean.roaster?.name}
					<div class="text-sm text-emphasis mt-0.5 flex items-center gap-1">
						<Icon name="store" class="w-3 h-3" />{bean.roaster.name}
					</div>
				{/if}
				<div class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1">
					{#if bean.origin}<span class="flex items-center gap-1"><Icon name="mapPin" class="w-3 h-3" />{bean.origin}</span>{/if}
					{#if bean.roast_level}<span class="flex items-center gap-1"><Icon name="flame" class="w-3 h-3" />{bean.roast_level}</span>{/if}
					{#if bean.variety}<span class="flex items-center gap-1"><Icon name="leaf" class="w-3 h-3" />{bean.variety}</span>{/if}
					{#if bean.process}<span class="flex items-center gap-1"><Icon name="sprout" class="w-3 h-3" />{bean.process}</span>{/if}
					{#if bean.rating}<span class="flex items-center gap-1"><Icon name="star" class="w-3 h-3 text-amber-500" />{bean.rating}/10</span>{/if}
				</div>
				{#if bean.description}
					<div class="mt-2 text-sm text-secondary italic line-clamp-2">"{bean.description}"</div>
				{/if}
				{#if bean.link && safeWebsiteURL(bean.link)}
					<div class="mt-2 text-xs text-muted flex items-center gap-1">
						<Icon name="link" class="w-3 h-3" />
						<a href={safeWebsiteURL(bean.link)} target="_blank" rel="noopener noreferrer" class="text-secondary hover:underline">{safeWebsiteURL(bean.link)}</a>
					</div>
				{/if}
			</div>
		{:else if roaster}
			<div class="feed-content-box-sm">
				<div class="font-bold text-primary text-base">{roaster.name}</div>
				<div class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1">
					{#if roaster.location}<span class="flex items-center gap-1"><Icon name="mapPin" class="w-3 h-3" />{roaster.location}</span>{/if}
					{#if roaster.website && safeWebsiteURL(roaster.website)}
						<span class="flex items-center gap-1"><Icon name="globe" class="w-3 h-3" /><a href={safeWebsiteURL(roaster.website)} target="_blank" rel="noopener noreferrer" class="text-secondary hover:underline">{safeWebsiteURL(roaster.website)}</a></span>
					{/if}
				</div>
			</div>
		{:else if grinder}
			<div class="feed-content-box-sm">
				<div class="font-bold text-primary text-base">{grinder.name}</div>
				<div class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1">
					{#if grinder.grinder_type}<span class="flex items-center gap-1"><Icon name="tag" class="w-3 h-3" />{grinder.grinder_type}</span>{/if}
					{#if grinder.burr_type}<span class="flex items-center gap-1"><Icon name="disc" class="w-3 h-3" />{grinder.burr_type}</span>{/if}
				</div>
				{#if grinder.notes}<div class="mt-2 text-sm text-secondary italic">"{grinder.notes}"</div>{/if}
			</div>
		{:else if brewer}
			<div class="feed-content-box-sm">
				<div class="font-bold text-primary text-base">{brewer.name}</div>
				{#if brewer.brewer_type}
					<div class="text-xs text-muted mt-1"><span class="flex items-center gap-1"><Icon name="brewer" class="w-3 h-3" />{brewer.brewer_type}</span></div>
				{/if}
				{#if brewer.description}<div class="mt-2 text-sm text-secondary italic">"{brewer.description}"</div>{/if}
			</div>
		{:else if recipe}
			<div class="feed-content-box-sm">
				<div class="text-base mb-2"><span class="font-bold text-primary">{recipe.name}</span></div>
				<div class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1">
					{#if recipe.coffee_amount > 0}<span class="flex items-center gap-1"><Icon name="coffee" class="w-3 h-3" />{recipe.coffee_amount.toFixed(1)}g</span>{/if}
					{#if recipe.water_amount > 0}<span class="flex items-center gap-1"><Icon name="droplet" class="w-3 h-3" />{recipe.water_amount.toFixed(1)}g</span>{/if}
					{#if recipe.brewer_obj?.name}<span class="flex items-center gap-1"><Icon name="brewer" class="w-3 h-3" />{recipe.brewer_obj.name}</span>{/if}
					{#if !recipe.brewer_obj?.name && recipe.brewer_type}<span class="flex items-center gap-1"><Icon name="brewer" class="w-3 h-3" />{recipe.brewer_type}</span>{/if}
				</div>
				{#if recipe.notes}<div class="mt-2 text-sm text-secondary italic">"{recipe.notes}"</div>{/if}
			</div>
		{/if}
	</div>

	<!-- Action bar -->
	{#if item.subject_uri && item.subject_cid}
		<ActionBar
			subjectURI={item.subject_uri}
			subjectCID={item.subject_cid}
			isLiked={item.is_liked_by_viewer}
			likeCount={item.like_count}
			commentCount={item.comment_count}
			shareURL={url}
			shareTitle={title}
			shareText={`Check out this ${actionNoun()} by ${item.author.display_name || displayHandle(item.author.handle)} on Arabica`}
			isOwner={item.is_owner}
			{editURL}
			{deleteURL}
			deleteRedirect="/my-coffee"
			viewURL={url}
			{isAuthenticated}
			isModerator={item.is_moderator}
			canHideRecord={item.can_hide_record}
			canBlockUser={item.can_block_user}
			isRecordHidden={item.is_record_hidden}
			authorDID={item.author_did}
		/>
	{/if}

	{#if footer}
		<div class="mt-1">
			{@render footer()}
		</div>
	{/if}
</div>
