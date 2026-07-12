<script lang="ts">
	import type { Snippet } from "svelte";
	type Props = { title: string; description?: string; children: Snippet };
	let { title, description = "", children }: Props = $props();
	let id = $derived(`form-section-${title.toLowerCase().replace(/[^a-z0-9]+/g, "-")}`);
</script>

<section class="form-section" aria-labelledby={id}>
	<header><h2 id={id}>{title}</h2>{#if description}<p>{description}</p>{/if}</header>
	<div class="form-section__body">{@render children()}</div>
</section>

<style>
	.form-section { padding: 1.4rem 0 1.75rem; border-top: 1px solid var(--card-border); }
	.form-section:first-child { border-top: 2px solid var(--text-secondary); }
	header { display: grid; grid-template-columns: minmax(8rem, .65fr) minmax(0, 1.35fr); gap: 1rem; margin-bottom: 1rem; }
	h2 { margin: 0; color: var(--text-primary); font-family: var(--font-display); font-size: 1.15rem; font-weight: 600; }
	p { margin: 0; color: var(--text-muted); font-size: .78rem; line-height: 1.5; }
	.form-section__body { min-width: 0; }
	@media (max-width: 640px) { header { grid-template-columns: 1fr; gap: .35rem; } }
</style>
