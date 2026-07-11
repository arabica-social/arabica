<script lang="ts">
	import Icon from "$lib/components/Icon.svelte";
	import { safeWebsiteURL } from "$lib/utils/format";
	import type { Vendor } from "$lib/types/generated/oolong_entities";

	let { vendor }: { vendor: Vendor } = $props();

	let safeLink = $derived(safeWebsiteURL(vendor.website));
</script>

<div class="record-label p-4">
	<div class="label-detail">
		<span class="detail-label">
			<span class="inline-flex items-center gap-1">
				<Icon name="mapPin" class="w-4 h-4 text-red-400" />
				Location
			</span>
		</span>
		{#if vendor.location}
			<span class="detail-value-lg">{vendor.location}</span>
		{:else}
			<span class="text-sm text-faint">—</span>
		{/if}
	</div>
	{#if safeLink}
		<div class="label-detail">
			<span class="detail-label">Website</span>
			<a href={safeLink} target="_blank" rel="noopener noreferrer" class="detail-value hover:underline">
				{safeLink}
			</a>
		</div>
	{/if}
	{#if vendor.description}
		<div class="label-detail">
			<span class="detail-label">Notes</span>
			<span class="detail-value whitespace-pre-wrap">{vendor.description}</span>
		</div>
	{/if}
</div>
