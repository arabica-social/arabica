<script lang="ts">
	import Icon from "$lib/components/Icon.svelte";
	import { displayHandle } from "$lib/stores/session";
	import { goto } from "$app/navigation";
	import type { PageData } from "./$types";
	import type { OnboardingResponse, ReadinessStatus } from "$lib/types/api";

	let { data }: { data: PageData } = $props();

	// svelte-ignore state_referenced_locally
	let onboarding = $state<OnboardingResponse | null>(data.onboarding);

	// svelte-ignore state_referenced_locally
	$effect(() => {
		onboarding = data.onboarding;
	});

	function stampState(done: boolean, current: boolean): string {
		if (done) return "done";
		if (current) return "current";
		return "todo";
	}

	function nextRequired(r: ReadinessStatus): string {
		if (!r.HasBrewer) return "brewer";
		if (!r.HasRoaster) return "roaster";
		if (!r.HasBean) return "bean";
		return "";
	}

	function ready(r: ReadinessStatus): boolean {
		return r.HasBean && r.HasBrewer && r.HasRoaster;
	}

	function stepsRemaining(r: ReadinessStatus): number {
		let n = 0;
		if (!r.HasBrewer) n++;
		if (!r.HasRoaster) n++;
		if (!r.HasBean) n++;
		return n;
	}

	type Station = {
		no: string;
		kind: string;
		title: string;
		hint: string;
		required: boolean;
		done: boolean;
		items: string[];
		optional?: boolean;
	};

	let stations = $derived<Station[]>([
		{
			no: "01", kind: "brewer", title: "Brewer",
			hint: "Espresso machine, V60, French press — whatever you brew with.",
			required: true,
			done: onboarding?.readiness.HasBrewer ?? false,
			items: (onboarding?.brewers ?? []).map((b) => b.name),
		},
		{
			no: "02", kind: "roaster", title: "Roaster",
			hint: "Who roasted the beans. Pick a favorite local roaster to start.",
			required: true,
			done: onboarding?.readiness.HasRoaster ?? false,
			items: (onboarding?.roasters ?? []).map((r) => r.name),
		},
		{
			no: "03", kind: "bean", title: "Bean",
			hint: "Your current bag — single origin, blend, anything you're drinking.",
			required: true,
			done: onboarding?.readiness.HasBean ?? false,
			items: (onboarding?.beans ?? []).map((b) => b.name || b.origin),
		},
		{
			no: "04", kind: "grinder", title: "Grinder",
			hint: "Optional — skip this if you brew with pre-ground beans.",
			required: false,
			done: (onboarding?.grinders ?? []).length > 0,
			items: (onboarding?.grinders ?? []).map((g) => g.name),
			optional: true,
		},
	]);

	function addEntity(kind: string) {
		// Navigate to the add records page or trigger a modal.
		// For now, link to /add (the add records surface).
		goto(`/add?entity=${kind}`);
	}

	function logBrew() {
		goto("/brews/new");
	}
</script>

<svelte:head>
	<title>Get Started - Arabica</title>
</svelte:head>

<div class="onboarding-wrap">
	<header class="onboarding-hero">
		<span class="onboarding-eyebrow">welcome to arabica</span>
		<h1 class="onboarding-title">Let's get you started.</h1>
		<p class="onboarding-lede">
			Add a <strong>brewer</strong>, a <strong>roaster</strong>, and your first bag of <strong>beans</strong> — that's all you need to start logging brews. Takes about two minutes.
		</p>
	</header>

	{#if data.error}
		<div class="card card-inner text-center py-8">
			<p class="text-secondary mb-4">{data.error}</p>
			<a href="/login" class="btn-primary">Log In</a>
		</div>
	{:else if onboarding}
		<section class="onboarding-card">
			<!-- Progress strip -->
			<div class="onboarding-progress" role="group" aria-label="Setup progress">
				{#each stations as station, i (station.kind)}
					<div class="stamp" data-state={stampState(station.done, nextRequired(onboarding.readiness) === station.kind)}>
						<span class="stamp-mark" aria-hidden="true">
							{#if station.done}
								<Icon name="shieldCheck" class="w-5 h-5 text-green-600" />
							{:else}
								{station.no}
							{/if}
						</span>
					</div>
					{#if i < stations.length - 1}
						<div class="stamp-line" data-state={stampState(station.done, false)}></div>
					{/if}
				{/each}
			</div>

			<!-- Stations -->
			<div class="stations">
				{#each stations as station (station.kind)}
					<div class="station" data-state={stampState(station.done, nextRequired(onboarding.readiness) === station.kind)}>
						<div class="station-header">
							<div class="station-number">{station.no}</div>
							<div class="station-info">
								<h3 class="station-title">
									{station.title}
									{#if !station.required}<span class="station-optional">optional</span>{/if}
								</h3>
								<p class="station-hint">{station.hint}</p>
							</div>
							<div class="station-actions">
								{#if station.done}
									<span class="badge badge-success">✓ Done</span>
								{/if}
								<button type="button" class="btn-secondary text-sm" onclick={() => addEntity(station.kind)}>
									{station.done ? "Add another" : `Add ${station.title}`}
								</button>
							</div>
						</div>
						{#if station.items.length > 0}
							<div class="station-items">
								{#each station.items as item}
									<span class="station-item">{item}</span>
								{/each}
							</div>
						{/if}
					</div>
				{/each}
			</div>

			<!-- Ready panel -->
			{#if ready(onboarding.readiness)}
				<div class="ready-panel">
					<div class="ready-panel-content">
						<h3 class="ready-title">You're ready to brew! ☕</h3>
						<p class="ready-text">All set. Log your first brew and start building your coffee story.</p>
					</div>
					<button type="button" class="btn-primary text-lg px-8 py-3" onclick={logBrew}>
						Log Your First Brew
					</button>
				</div>
			{:else}
				<div class="ready-panel ready-panel-pending">
					<p class="text-sm text-muted">
						{stepsRemaining(onboarding.readiness)} step{stepsRemaining(onboarding.readiness) !== 1 ? "s" : ""} to go before you can log a brew.
					</p>
				</div>
			{/if}
		</section>
	{/if}
</div>
