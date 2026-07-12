<script lang="ts">
	import type { Snippet } from "svelte";
	import BackButton from "./BackButton.svelte";

	type Props = {
		title: string;
		eyebrow?: string;
		description?: string;
		showBack?: boolean;
		actions?: Snippet;
	};

	let { title, eyebrow = "", description = "", showBack = false, actions }: Props = $props();
</script>

<header class="ledger-header">
	<div class="ledger-header__identity">
		{#if showBack}<BackButton />{/if}
		<div>
			{#if eyebrow}<p>{eyebrow}</p>{/if}
			<h1>{title}</h1>
			{#if description}<div class="ledger-header__description">{description}</div>{/if}
		</div>
	</div>
	{#if actions}<div class="ledger-header__actions">{@render actions()}</div>{/if}
</header>

<style>
	.ledger-header { display: flex; align-items: end; justify-content: space-between; gap: 1.5rem; padding: .5rem 0 1.25rem; border-bottom: 1px solid var(--card-border); }
	.ledger-header__identity { display: flex; align-items: flex-start; gap: .75rem; min-width: 0; }
	p { margin: 0 0 .35rem; color: var(--text-muted); font-size: .65rem; font-weight: 700; letter-spacing: .14em; text-transform: uppercase; }
	h1 { margin: 0; color: var(--text-primary); font-family: var(--font-display); font-size: clamp(2rem, 4vw, 3rem); font-weight: 600; line-height: 1; letter-spacing: -.025em; }
	.ledger-header__description { max-width: 60ch; margin-top: .65rem; color: var(--text-secondary); font-size: .9rem; line-height: 1.6; }
	.ledger-header__actions { flex: 0 0 auto; }
	@media (max-width: 640px) { .ledger-header { align-items: stretch; flex-direction: column; } .ledger-header__actions { width: 100%; } }
</style>
