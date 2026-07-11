<script lang="ts">
	// Redirects to /my-coffee via +page.ts load. This component only renders
	// if the redirect is somehow bypassed.
	import type { PageData } from "./$types";
	let { data }: { data: PageData } = $props();
</script>

<div class="page-container-sm">
	<div class="card card-inner text-center py-8">
		<p class="text-secondary">Redirecting to <a href="/my-coffee">My Coffee</a>…</p>
	</div>
</div>
