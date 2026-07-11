<script lang="ts">
	import { onMount } from "svelte";

	type Suggestion = {
		name: string;
		source_uri: string;
		fields?: Record<string, string>;
		count?: number;
	};

	type Props = {
		endpoint: string;
		placeholder: string;
		name: string;
		origin?: string;
		roastLevel?: string;
		process?: string;
		link?: string;
		sourceRef?: string;
		inputId?: string;
		ariaLabel?: string;
		ariaDescribedby?: string;
		error?: string;
		oninput?: () => void;
	};

	let {
		endpoint,
		placeholder,
		name = $bindable(),
		origin = $bindable(),
		roastLevel = $bindable(),
		process = $bindable(),
		link = $bindable(),
		sourceRef = $bindable(),
		inputId,
		ariaLabel,
		ariaDescribedby,
		error = "",
		oninput,
	}: Props = $props();

	let query = $state(name);
	let originalName = $state("");
	let suggestions = $state<Suggestion[]>([]);
	let showSuggestions = $state(false);
	let searchTimer: ReturnType<typeof setTimeout> | undefined;
	let blurTimer: ReturnType<typeof setTimeout> | undefined;

	// Keep query in sync if parent updates name (e.g. form reset)
	let lastExternalName = name;
	$effect(() => {
		if (name !== lastExternalName && name !== query) {
			query = name;
		}
		lastExternalName = name;
	});

	function resetSourceIfEdited() {
		if (!originalName || query.trim().toLowerCase() === originalName.toLowerCase()) {
			return;
		}
		sourceRef = "";
		originalName = "";
	}

	async function search() {
		resetSourceIfEdited();
		const q = query.trim();
		if (q.length < 2) {
			suggestions = [];
			showSuggestions = false;
			return;
		}
		try {
			const response = await fetch(
				`${endpoint}?q=${encodeURIComponent(q)}&limit=10`,
				{ credentials: "same-origin" },
			);
			if (!response.ok) return;
			const data: unknown = await response.json();
			suggestions = Array.isArray(data) ? (data as Suggestion[]) : [];
			showSuggestions = suggestions.length > 0;
		} catch {
			// Suggestions are optional.
		}
	}

	function onInput() {
		name = query;
		oninput?.();
		window.clearTimeout(searchTimer);
		searchTimer = window.setTimeout(() => {
			void search();
		}, 300);
	}

	function onFocus() {
		if (suggestions.length > 0) {
			showSuggestions = true;
		}
	}

	function onBlur() {
		window.clearTimeout(blurTimer);
		blurTimer = window.setTimeout(() => {
			showSuggestions = false;
		}, 200);
	}

	function selectSuggestion(suggestion: Suggestion) {
		const fields = suggestion.fields || {};
		query = suggestion.name;
		name = suggestion.name;
		sourceRef = suggestion.source_uri;
		originalName = suggestion.name;
		showSuggestions = false;

		if (fields.origin) origin = fields.origin;
		if (fields.roastLevel) roastLevel = fields.roastLevel;
		if (fields.process) process = fields.process;
		if (fields.link) link = fields.link;
	}

	function secondaryText(suggestion: Suggestion): string {
		const fields = suggestion.fields || {};
		return fields.origin || "";
	}

	onMount(() => {
		return () => {
			window.clearTimeout(searchTimer);
			window.clearTimeout(blurTimer);
		};
	});
</script>

<div class="relative">
	<input
		id={inputId}
		type="text"
		{placeholder}
		bind:value={query}
		oninput={onInput}
		onblur={onBlur}
		onfocus={onFocus}
		autocomplete="off"
		class="w-full form-input-lg"
		aria-label={ariaLabel}
		aria-invalid={error !== ""}
		aria-describedby={ariaDescribedby}
	/>

	{#if showSuggestions && suggestions.length > 0}
		<div class="suggestions-dropdown">
			{#each suggestions as suggestion}
				<button
					type="button"
					class="suggestions-item"
					onmousedown={(event) => {
						event.preventDefault();
						selectSuggestion(suggestion);
					}}
				>
					<span class="font-medium">{suggestion.name}</span>
					{#if secondaryText(suggestion)}
						<span class="text-xs text-faint">{secondaryText(suggestion)}</span>
					{/if}
					{#if (suggestion.count || 0) > 1}
						<span class="text-xs text-placeholder">{suggestion.count} users</span>
					{/if}
				</button>
			{/each}
		</div>
	{/if}
</div>
