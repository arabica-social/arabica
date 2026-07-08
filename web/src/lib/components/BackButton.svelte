<script lang="ts">
	// Simple back button using browser history. Ported from
	// internal/web/components/buttons.templ BackButton().
	function goBack() {
		if (window.history.length > 1) {
			window.history.back();
		} else {
			window.location.href = "/";
		}
	}
</script>

<button
	type="button"
	onclick={goBack}
	class="inline-flex items-center text-emphasis hover:text-primary font-medium transition-colors cursor-pointer"
	aria-label="Go back"
>
	<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
		<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"></path>
	</svg>
</button>
