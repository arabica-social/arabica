<script lang="ts">
	import type { Snippet } from "svelte";
	type Props = { children: Snippet; rail?: Snippet };
	let { children, rail }: Props = $props();
</script>

<div class="form-workspace">
	<main class="form-workspace__content">{@render children()}</main>
	{#if rail}<aside class="form-workspace__rail" aria-label="Form context">{@render rail()}</aside>{/if}
</div>

<style>
	.form-workspace { width: 100%; max-width: 72rem; margin-inline: auto; padding-inline: clamp(.5rem, 2.5vw, 1.5rem); display: grid; grid-template-columns: minmax(0, 1fr); gap: 2rem; }
	.form-workspace__content { min-width: 0; }
	.form-workspace__rail { min-width: 0; }
	@media (min-width: 960px) { .form-workspace { grid-template-columns: minmax(0, 2.4fr) minmax(14rem, .8fr); gap: clamp(2rem, 4vw, 4rem); } .form-workspace__rail { position: sticky; top: 5rem; align-self: start; } }
</style>
