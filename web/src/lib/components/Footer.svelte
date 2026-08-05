<script lang="ts">
	import { app } from "$lib/stores/session";
	import { definitionFor } from "$lib/app/definitions";

	let {
		brandName = "Arabica",
		tagline = "Your brew, your data",
	}: { brandName?: string; tagline?: string } = $props();

	let feedbackUrl = $derived(definitionFor($app).feedbackUrl);
</script>

<footer class="site-footer mt-auto" style="background: var(--footer-bg); border-top: 1px solid var(--footer-border);">
	<div class="site-footer-shell container mx-auto px-4 py-8">
		<div class="site-footer-row flex flex-col md:flex-row justify-between items-center space-y-4 md:space-y-0">
			<div class="site-footer-brand text-sm text-emphasis">
				<span class="font-semibold text-primary">{brandName}</span> - {tagline}
			</div>
			<nav class="site-footer-nav flex flex-wrap justify-center gap-6 text-sm">
				<a href="/about" class="text-emphasis hover:text-primary transition-colors">About</a>
				<a href={feedbackUrl} target="_blank" rel="noopener noreferrer" class="text-emphasis hover:text-primary transition-colors">Feedback</a>
				<a href="/terms" class="text-emphasis hover:text-primary transition-colors">Terms of Service</a>
				<a href="https://tangled.org/arabica.social/arabica" target="_blank" rel="noopener noreferrer" class="text-emphasis hover:text-primary transition-colors">Source Code</a>
				<a href="/atproto" class="text-emphasis hover:text-primary transition-colors">AT Protocol</a>
			</nav>
		</div>
	</div>
</footer>
