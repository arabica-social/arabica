<script lang="ts">
	import type { FeedFilterTab } from "../types/feed";

	type Props = {
		typeFilter: string;
		sort: string;
		loading: boolean;
		onType: (value: string) => void;
		onSort: (value: string) => void;
		tabs?: FeedFilterTab[];
	};

	let { typeFilter, sort, loading, onType, onSort, tabs = [] }: Props = $props();

	// Server-provided tabs are app-scoped (arabica vs oolong). Fall back to a
	// minimal "All" tab when the response hasn't loaded yet.
	let activeTabs = $derived(tabs.length > 0 ? tabs : [{ label: "All", value: "" }]);

	function pillClass(tab: string, active: string): string {
		return typeFilter === tab ? "filter-pill-active" : "filter-pill";
	}

	function sortClass(s: string): string {
		if (s === "recent" && (sort === "" || sort === "recent")) return "filter-pill-active";
		return sort === s ? "filter-pill-active" : "filter-pill";
	}
</script>

<div class="mb-5 flex flex-wrap items-center justify-between gap-2" aria-busy={loading}>
	<div class="flex flex-wrap gap-1" role="group" aria-label="Feed filters">
		{#each activeTabs as tab (tab.value)}
			<button
				type="button"
				class={pillClass(tab.value, typeFilter)}
				aria-pressed={typeFilter === tab.value ? "true" : "false"}
				data-tab={tab.value}
				disabled={loading}
				onclick={() => onType(tab.value)}
			>
				{tab.label}
			</button>
		{/each}
	</div>
	<div class="flex items-center gap-1 flex-shrink-0">
		<button
			type="button"
			class={sortClass("recent")}
				aria-pressed={sort === "" || sort === "recent" ? "true" : "false"}
			disabled={loading}
			onclick={() => onSort("recent")}
		>
			New
		</button>
		<button
			type="button"
			class={sortClass("popular")}
				aria-pressed={sort === "popular" ? "true" : "false"}
				disabled={loading}
			onclick={() => onSort("popular")}
		>
			Popular
		</button>
	</div>
</div>
