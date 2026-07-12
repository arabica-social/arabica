<script lang="ts">
	import type { Snippet } from "svelte";
	type Props = { title: string; eyebrow?: string; children: Snippet; lead?: boolean };
	let { title, eyebrow = "", children, lead = false }: Props = $props();
</script>

<section class:lead class="rail-section">
	{#if eyebrow}<p class="rail-section__eyebrow">{eyebrow}</p>{/if}
	<h2>{title}</h2>
	<div class="rail-section__body">{@render children()}</div>
</section>

<style>
	.rail-section { padding: 1rem 0 1.2rem; border-top: 1px solid var(--card-border); }
	.rail-section.lead { border-top: 2px solid var(--text-secondary); }
	.rail-section__eyebrow { margin: 0 0 .4rem; color: var(--text-muted); font-size: .65rem; font-weight: 700; letter-spacing: .14em; text-transform: uppercase; }
	h2 { margin: 0 0 .55rem; color: var(--text-primary); font-family: var(--font-display); font-size: 1.05rem; font-weight: 600; line-height: 1.3; }
	.rail-section__body { color: var(--text-muted); font-size: .76rem; line-height: 1.55; }
	.rail-section__body :global(p) { margin: 0 0 .7rem; }
</style>
