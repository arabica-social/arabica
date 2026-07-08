<script lang="ts">
	import BackButton from "$lib/components/BackButton.svelte";
	import type { PageData } from "./$types";

	let { data }: { data: PageData } = $props();

	function locationPin() {
		return `<svg class="w-3 h-3" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M15 10.5a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z"></path><path stroke-linecap="round" stroke-linejoin="round" d="M19.5 10.5c0 7.142-7.5 11.25-7.5 11.25S4.5 17.642 4.5 10.5a7.5 7.5 0 1 1 15 0Z"></path></svg>`;
	}
</script>

<svelte:head>
	<title>Create an Atmosphere Account - Arabica</title>
	<meta name="description" content="Create an Atmosphere account — one account for the entire AT Protocol ecosystem. Arabica, Bluesky, Tangled, and more." />
	<meta property="og:title" content="Create an Atmosphere Account - Arabica" />
	<meta property="og:type" content="website" />
</svelte:head>

<div class="page-container-md">
	<div class="flex items-center gap-3 mb-8">
		<BackButton />
		<h1 class="text-4xl font-bold text-primary">Create an Atmosphere Account</h1>
	</div>
	<p class="text-secondary leading-relaxed mb-2">
		An Atmosphere account is your passport to Arabica and the wider <a href="/atproto" class="link-bold">AT Protocol</a> ecosystem.
		One account works across every compatible app — no more creating new logins.
	</p>
	<p class="text-sm text-muted mb-6">
		Choose a provider below. Your data is portable — you can move to a different provider at any time.
	</p>

	{#if data.error}
		<div class="mb-6 rounded-lg border border-red-300 bg-red-50 p-4 text-red-800 text-sm">
			{data.error}
		</div>
	{/if}

	{#if data.loadFailed}
		<div class="mb-6 rounded-lg border border-amber-300 bg-amber-50 p-4 text-amber-800 text-sm">
			Could not load the provider list. Please try again in a moment.
		</div>
	{/if}

	{#each data.categories as cat (cat.title)}
		<div class="mb-6">
			<h2 class="text-sm font-medium text-muted uppercase tracking-wider mb-3">{cat.title}</h2>
			<p class="text-sm text-faint mb-3">{cat.description}</p>
			<div class="space-y-3">
				{#each cat.providers as p (p.url)}
					<div class="card card-inner">
						<p class="flex items-baseline gap-2 text-lg font-semibold text-primary">
							{p.name}
							<span class={`relative -top-px inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium bg-${p.badge_color}-100 text-${p.badge_color}-800`}>
								{p.badge}
							</span>
						</p>
						<p class="text-sm text-emphasis">
							{p.description}
							{#if p.operator_name}
								{" hosted by "}
								<a href={p.operator_url} target="_blank" rel="noopener noreferrer" class="link-bold">{p.operator_name}</a>.
							{/if}
						</p>
						<p class="flex items-center gap-1 text-xs text-faint">
							{p.domain} ·
							<!-- eslint-disable-next-line svelte/no-at-html-tags -->
							{@html locationPin()}
							{p.location}
						</p>
						<div class="mt-3">
							{#if p.signup_url}
								<a href={p.signup_url} target="_blank" rel="noopener noreferrer" class="btn-primary">
									Create Account
								</a>
							{:else}
								<form method="POST" action="/join/create">
									<input type="hidden" name="pds_url" value={p.url} />
									<button type="submit" class="btn-primary">Create Account</button>
								</form>
							{/if}
						</div>
					</div>
				{/each}
			</div>
		</div>
	{/each}

	<div class="text-sm text-faint mb-6">
		<strong class="text-emphasis">Self-hosted:</strong> Technical users can <a href="https://atproto.com/guides/self-hosting" target="_blank" rel="noopener noreferrer" class="link">run their own provider</a> for full control over their data.
	</div>
	<p class="text-sm text-muted text-center">
		Already have an Atmosphere account?
		<a href="/login" class="link-bold">Log in here</a>.
	</p>
</div>
