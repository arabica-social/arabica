<script lang="ts">
	import { onMount } from "svelte";
	import EntityCombo from "./EntityCombo.svelte";
	import Field from "./BrewFormField.svelte";
	import PoursEditor from "./PoursEditor.svelte";
	import FormWorkspace from "./FormWorkspace.svelte";
	import LedgerHeader from "./LedgerHeader.svelte";
	import FormSection from "./FormSection.svelte";
	import RailSection from "./RailSection.svelte";
	import { appCache } from "../stores/appCache";
	import { session, warnIfSessionExpired } from "../stores/session";
	import { pushToast } from "../stores/toasts";
	import { goto } from "$app/navigation";
	import { createBrew, updateBrew, type BrewInput } from "../api/entities";
	import type { Brew, Recipe } from "../types/entity_view";

	type Pour = { water: string; time: string };

	type Props = {
		brew: Brew | null;
		recipeRKey?: string;
		recipeOwnerDID?: string;
		isEdit: boolean;
	};

	let { brew, recipeRKey = "", recipeOwnerDID = "", isEdit }: Props = $props();

	let cachedData = $state<Record<string, unknown>>({});
	let activeRecipe = $state<Recipe | null>(null);
	let recipeExpanded = $state(false);
	let recipeRatio = $state("");
	let recipePours = $state<Pour[]>([]);
	let brewerCategory = $state("");

	// svelte-ignore state_referenced_locally
	let recipeRKeyValue = $state(recipeRKey);
	let recipeLabel = $state("");
	// svelte-ignore state_referenced_locally
	let recipeOwner = $state(recipeOwnerDID);
	let beanRKey = $state("");
	let beanLabel = $state("");
	let grinderRKey = $state("");
	let grinderLabel = $state("");
	let brewerRKey = $state("");
	let brewerLabel = $state("");
	let coffeeAmount = $state("");
	let waterAmount = $state("");
	let grindSize = $state("");
	let temperature = $state("");
	let timeSeconds = $state("");
	let tastingNotes = $state("");
	let rating = $state("5");
	let method = $state("");
	let pours = $state<Pour[]>([]);
	let espressoYieldWeight = $state("");
	let espressoPressure = $state("");
	let espressoPreInfusionSeconds = $state("");
	let pouroverBloomWater = $state("");
	let pouroverBloomSeconds = $state("");
	let pouroverDrawdownSeconds = $state("");
	let pouroverBypassWater = $state("");
	let pouroverFilter = $state("");
	let submitting = $state(false);

	function normalizeBrewerCategory(raw: string): string {
		const lower = (raw || "").toLowerCase().trim();
		if (["pourover", "espresso", "immersion", "mokapot", "coldbrew", "cupping", "other"].includes(lower)) return lower;
		if (["pour-over", "pour over", "dripper"].includes(lower)) return "pourover";
		if (["espresso machine", "lever espresso", "lever espresso machine"].includes(lower)) return "espresso";
		if (["french press", "aeropress", "siphon", "clever", "clever dripper"].includes(lower)) return "immersion";
		return "";
	}

	function mustBePositive(value: string): boolean {
		if (value === null || value === "") return false;
		const parsed = Number(value);
		return !Number.isFinite(parsed) || parsed <= 0;
	}

	let coffeeAmountError = $derived(mustBePositive(coffeeAmount));
	let waterAmountError = $derived(mustBePositive(waterAmount));
	let temperatureError = $derived(mustBePositive(temperature));
	let timeSecondsError = $derived(mustBePositive(timeSeconds));

	type ComboType = "recipe" | "bean" | "grinder" | "brewer";

	function handleComboChange(type: ComboType, detail: Record<string, unknown>) {
		const rkey = (detail.rkey as string) ?? "";
		const entity = detail.entity as Record<string, unknown> | undefined;
		if (!rkey) {
			clearCombo(type);
			return;
		}
		if (type === "recipe") {
			recipeRKeyValue = rkey;
			recipeOwner = (detail.owner as string) ?? (entity?.author_did as string) ?? "";
			void applyRecipe(rkey, recipeOwner);
			return;
		}
		if (type === "bean") { beanRKey = rkey; return; }
		if (type === "grinder") { grinderRKey = rkey; return; }
		if (type === "brewer") {
			brewerRKey = rkey;
			brewerCategory = normalizeBrewerCategory((entity?.brewer_type as string) ?? "");
		}
	}

	function clearCombo(type: ComboType) {
		if (type === "recipe") { recipeRKeyValue = ""; recipeLabel = ""; activeRecipe = null; recipeOwner = ""; recipeExpanded = false; recipeRatio = ""; recipePours = []; }
		if (type === "bean") { beanRKey = ""; beanLabel = ""; }
		if (type === "grinder") { grinderRKey = ""; grinderLabel = ""; }
		if (type === "brewer") { brewerRKey = ""; brewerLabel = ""; brewerCategory = ""; }
	}

	async function applyRecipe(selectedRKey: string, selectedOwner = "") {
		if (!selectedRKey) { activeRecipe = null; recipeOwner = ""; recipeExpanded = false; return; }
		recipeOwner = selectedOwner;
		const ownerQuery = recipeOwner ? `?owner=${encodeURIComponent(recipeOwner)}` : "";
		try {
			const res = await fetch(`/api/recipes/${selectedRKey}${ownerQuery}`, { credentials: "same-origin" });
			if (!res.ok) return;
			const recipe = (await res.json()) as Recipe;
			activeRecipe = recipe;
			recipeExpanded = false;
			coffeeAmount = recipe.coffee_amount > 0 ? String(Math.round(recipe.coffee_amount)) : "";
			waterAmount = recipe.water_amount > 0 ? String(Math.round(recipe.water_amount)) : "";
			recipeRatio = formatRatio(recipe.coffee_amount, recipe.water_amount);
			recipePours = (recipe.pours ?? []).map((p) => ({ water: String(p.water_amount ?? ""), time: String(p.time_seconds ?? "") }));
			pours = recipePours.map((pour) => ({ ...pour }));
			// Derive the brewer from the recipe so the (hidden while collapsed,
			// visible after Adjust) brewer combo and the posted brewer_rkey reflect
			// the recipe's brewer rather than staying empty.
			const recipeBrewerType = recipe.brewer_type || recipe.brewer_obj?.brewer_type || "";
			if (recipeBrewerType) brewerCategory = normalizeBrewerCategory(recipeBrewerType);
			if (recipe.brewer_rkey) {
				brewerRKey = recipe.brewer_rkey;
				brewerLabel = recipe.brewer_obj?.name ?? "";
			}
			// For pour-over recipes, the bloom mirrors the first pour: its water
			// becomes the bloom amount and its time becomes the bloom duration.
			if (brewerCategory === "pourover" && pours.length > 0) {
				const firstPourWater = positiveNumber(pours[0].water);
				if (firstPourWater !== null) pouroverBloomWater = String(firstPourWater);
				const firstPourTime = positiveNumber(pours[0].time);
				if (firstPourTime !== null) pouroverBloomSeconds = String(firstPourTime);
			}
		} catch {
			// Ignore — user can fill manually.
		}
	}

	function formatRatio(coffee: number, water: number): string {
		if (coffee <= 0 || water <= 0) return "";
		return String(Number((water / coffee).toFixed(2)));
	}

	function positiveNumber(value: string): number | null {
		const parsed = Number(value);
		return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
	}

	function roundedAmount(value: number): string {
		return String(Math.round(value));
	}

	function scalePoursToWater(targetWater: number) {
		const pourTotal = recipePours.reduce((total, pour) => total + (positiveNumber(pour.water) ?? 0), 0);
		if (pourTotal <= 0) return;

		const lastPourIndex = recipePours.reduce(
			(lastIndex, pour, index) => (positiveNumber(pour.water) === null ? lastIndex : index),
			-1,
		);
		if (lastPourIndex === -1) return;

		let scaledTotal = 0;
		const scaled = recipePours.map((pour) => {
			const water = positiveNumber(pour.water);
			if (water === null) return pour;
			const scaledWater = Math.round((water / pourTotal) * targetWater);
			scaledTotal += scaledWater;
			return { ...pour, water: String(scaledWater) };
		});
		const lastPour = scaled[lastPourIndex];
		scaled[lastPourIndex] = {
			...lastPour,
			water: String(Math.max(0, Number(lastPour.water) + targetWater - scaledTotal)),
		};
		pours = scaled;
		// Keep the pour-over bloom water in lockstep with the first pour while a
		// recipe is driving the pours. Bloom only tracks the first pour; later
		// manual edits to bloom water survive until the next scaling pass.
		if (activeRecipe && brewerCategory === "pourover" && scaled.length > 0) {
			const firstPourWater = positiveNumber(scaled[0].water);
			if (firstPourWater !== null) pouroverBloomWater = String(firstPourWater);
		}
	}

	function setRecipeMeasurements(nextCoffee: number, nextWater: number) {
		coffeeAmount = roundedAmount(nextCoffee);
		waterAmount = roundedAmount(nextWater);
		scalePoursToWater(Math.round(nextWater));
	}

	function adjustRecipeCoffee(value: string) {
		coffeeAmount = value;
		const coffee = positiveNumber(value);
		const ratio = positiveNumber(recipeRatio);
		if (coffee === null || ratio === null) return;
		setRecipeMeasurements(coffee, coffee * ratio);
	}

	function adjustRecipeWater(value: string) {
		waterAmount = value;
		const water = positiveNumber(value);
		const ratio = positiveNumber(recipeRatio);
		if (water === null || ratio === null) return;
		setRecipeMeasurements(water / ratio, water);
	}

	function adjustRecipeRatio(value: string) {
		recipeRatio = value;
		const ratio = positiveNumber(value);
		const coffee = positiveNumber(coffeeAmount);
		if (ratio === null || coffee === null) return;
		setRecipeMeasurements(coffee, coffee * ratio);
	}

	function showRecipeOverrides(): boolean {
		return !activeRecipe || recipeExpanded;
	}

	function recipeSummary(): string {
		if (!activeRecipe) return "";
		const parts: string[] = [];
		if (activeRecipe.coffee_amount > 0) parts.push(`${Math.round(activeRecipe.coffee_amount)}g coffee`);
		if (activeRecipe.water_amount > 0) parts.push(`${Math.round(activeRecipe.water_amount)}g water`);
		if (activeRecipe.pours?.length) parts.push(`${activeRecipe.pours.length} pours`);
		return parts.join(" · ");
	}

	function initializeFromBrew() {
		if (!brew) return;
		beanRKey = brew.bean_rkey ?? "";
		beanLabel = brew.bean ? `${brew.bean.name || brew.bean.origin} (${brew.bean.origin} - ${brew.bean.roast_level})` : "";
		grinderRKey = brew.grinder_rkey ?? "";
		grinderLabel = brew.grinder_obj?.name ?? "";
		brewerRKey = brew.brewer_rkey ?? "";
		brewerLabel = brew.brewer_obj?.name ?? "";
		recipeRKeyValue = brew.recipe_rkey ?? "";
		recipeLabel = brew.recipe_obj?.name ?? "";
		coffeeAmount = brew.coffee_amount > 0 ? String(brew.coffee_amount) : "";
		waterAmount = brew.water_amount > 0 ? String(brew.water_amount) : "";
		grindSize = brew.grind_size ?? "";
		temperature = brew.temperature > 0 ? String(brew.temperature) : "";
		timeSeconds = brew.time_seconds > 0 ? String(brew.time_seconds) : "";
		tastingNotes = brew.tasting_notes ?? "";
		rating = brew.rating > 0 ? String(brew.rating) : "5";
		method = brew.method ?? "";
		pours = (brew.pours ?? []).map((p) => ({ water: String(p.water_amount ?? ""), time: String(p.time_seconds ?? "") }));
		if (brew.espresso_params) {
			espressoYieldWeight = brew.espresso_params.yield_weight > 0 ? String(brew.espresso_params.yield_weight) : "";
			espressoPressure = brew.espresso_params.pressure > 0 ? String(brew.espresso_params.pressure) : "";
			espressoPreInfusionSeconds = brew.espresso_params.pre_infusion_seconds > 0 ? String(brew.espresso_params.pre_infusion_seconds) : "";
		}
		if (brew.pourover_params) {
			pouroverBloomWater = brew.pourover_params.bloom_water > 0 ? String(brew.pourover_params.bloom_water) : "";
			pouroverBloomSeconds = brew.pourover_params.bloom_seconds > 0 ? String(brew.pourover_params.bloom_seconds) : "";
			pouroverDrawdownSeconds = brew.pourover_params.drawdown_seconds > 0 ? String(brew.pourover_params.drawdown_seconds) : "";
			pouroverBypassWater = brew.pourover_params.bypass_water > 0 ? String(brew.pourover_params.bypass_water) : "";
			pouroverFilter = brew.pourover_params.filter ?? "";
		}
		brewerCategory = brew.espresso_params ? "espresso" : brew.pourover_params ? "pourover" : brew.brewer_obj?.brewer_type ?? "";
	}

	initializeFromBrew();

	onMount(() => {
		// Load app cache for EntityCombo.
		appCache.getData().then((data) => {
			if (data) cachedData = data;
		});
		// Auto-apply recipe from URL param on new brew.
		if (recipeRKeyValue && !brew) {
			void applyRecipe(recipeRKeyValue, recipeOwner);
		}
		// Proactively check whether the OAuth session is still resumable so we
		// can prompt re-authentication before a failed save, rather than after
		// the user has filled in the whole form.
		void warnIfSessionExpired();
	});

	async function submitForm(e: SubmitEvent) {
		e.preventDefault();
		if (submitting) return;
		submitting = true;

		const input: BrewInput = { bean_rkey: beanRKey };
		if (recipeRKeyValue) {
			input.recipe_rkey = recipeRKeyValue;
			if (recipeOwner) input.recipe_owner_did = recipeOwner;
		}
		if (method) input.method = method;
		if (coffeeAmount) input.coffee_amount = num(coffeeAmount);
		if (waterAmount) input.water_amount = num(waterAmount);
		if (grindSize) input.grind_size = grindSize;
		if (temperature) input.temperature = num(temperature);
		if (timeSeconds) input.time_seconds = num(timeSeconds);
		if (tastingNotes) input.tasting_notes = tastingNotes;
		if (rating) input.rating = num(rating);
		if (grinderRKey) input.grinder_rkey = grinderRKey;
		if (brewerRKey) input.brewer_rkey = brewerRKey;
		const poursInput = pours
			.filter((pour) => pour.water || pour.time)
			.map((pour) => ({ water_amount: num(pour.water), time_seconds: num(pour.time) }));
		if (poursInput.length > 0) input.pours = poursInput;
		if (espressoYieldWeight || espressoPressure || espressoPreInfusionSeconds) {
			input.espresso_params = {
				yield_weight: num(espressoYieldWeight),
				pressure: num(espressoPressure),
				pre_infusion_seconds: num(espressoPreInfusionSeconds),
			};
		}
		if (
			pouroverBloomWater ||
			pouroverBloomSeconds ||
			pouroverDrawdownSeconds ||
			pouroverBypassWater ||
			pouroverFilter
		) {
			input.pourover_params = {
				bloom_water: num(pouroverBloomWater),
				bloom_seconds: num(pouroverBloomSeconds),
				drawdown_seconds: num(pouroverDrawdownSeconds),
				bypass_water: num(pouroverBypassWater),
				filter: pouroverFilter,
			};
		}

		try {
			const data = isEdit
				? await updateBrew(fetch, brew?.rkey ?? "", input)
				: await createBrew(fetch, input);
			const savedBrew = data.brew;
			pushToast(isEdit ? "Brew updated!" : "Brew saved!");
			// The JSON envelope carries author_did at the top level (the Brew
			// record model has no author field); fall back to the session DID
			// for edits where the envelope may omit it.
			const actor = data.author_did ?? $session.did ?? "";
			const rkey = savedBrew.rkey ?? brew?.rkey ?? "";
			if (actor && rkey) {
				goto(`/brews/${actor}/${rkey}`);
			} else {
				goto("/my-coffee");
			}
		} catch (error) {
			console.error("Brew save failed:", error);
			pushToast("Failed to save brew");
		} finally {
			submitting = false;
		}
	}

	let submitLabel = $derived(isEdit ? "Update brew" : "Save brew");

	function num(value: string): number {
		const n = Number(value);
		return Number.isFinite(n) ? n : 0;
	}
	let coffeeValue = $derived(num(coffeeAmount));
	let waterValue = $derived(num(waterAmount));
	let ratio = $derived(coffeeValue > 0 && waterValue > 0 ? waterValue / coffeeValue : null);
	let brewerCategoryLabel = $derived(
		brewerCategory === "pourover"
			? "Pour-over"
			: brewerCategory === "espresso"
				? "Espresso"
				: brewerCategory === "immersion"
					? "Immersion"
					: brewerCategory === "mokapot"
						? "Moka pot"
						: brewerCategory === "coldbrew"
							? "Cold brew"
							: brewerCategory === "cupping"
								? "Cupping"
								: brewerCategory === "other"
									? "Other"
									: "",
	);
	let pourCount = $derived(pours.filter((p) => p.water !== "" || p.time !== "").length);
</script>

<FormWorkspace>
	<LedgerHeader
		title={isEdit ? "Edit Brew" : "New Brew"}
		eyebrow="Brew session"
		description="Record the recipe, equipment, measurements, and results."
		showBack={true}
	/>

	<form class="brew-form-sheet" novalidate onsubmit={submitForm}>
		<FormSection title="Recipe (optional)" description="Select a recipe to fill in its measurements and pours.">
			<div class="alert-warning px-3 py-2 mb-2 text-xs">
				Recipes are in early alpha. The recipe format may change, but your brew record will not.
			</div>
			<div class="combo-select">
				<EntityCombo
					entityType="recipe"
					inputName="recipe_rkey"
					apiEndpoint="/api/recipes"
					suggestEndpoint="/api/suggestions/recipes"
					placeholder="Search recipes..."
					sectionLabel="Your recipes"
					passthrough={true}
					allowCreate={false}
					bind:rkey={recipeRKeyValue}
					bind:label={recipeLabel}
					ariaLabel="Search recipes"
					onChange={(detail) => handleComboChange("recipe", detail)}
				/>
			</div>

			{#if activeRecipe}
				<div class="section-box">
					<div class="flex items-center justify-between gap-2">
						<p class="text-sm text-emphasis flex-1">{recipeSummary()}</p>
						<button type="button" onclick={() => (recipeExpanded = !recipeExpanded)} class="text-sm btn-secondary" aria-expanded={recipeExpanded}>
							{recipeExpanded ? "Done adjusting" : "Adjust"}
						</button>
					</div>
					{#if recipeExpanded}
						<div class="recipe-adjustment" aria-label="Recipe adjustment">
							<div class="recipe-adjustment-heading">
								<p class="form-label">Quick adjustment</p>
								<p class="text-helper">Set a dose, water, or ratio. The other measurement follows, and recipe pours scale with the water.</p>
							</div>
							<div class="recipe-adjustment-grid">
								<label for="recipe-adjustment-coffee">
									<span class="form-label">Coffee (g)</span>
									<input id="recipe-adjustment-coffee" type="number" value={coffeeAmount} oninput={(event) => adjustRecipeCoffee(event.currentTarget.value)} step="1" min="1" class="w-full form-input-lg" />
								</label>
								<label for="recipe-adjustment-ratio">
									<span class="form-label">Ratio (1:X)</span>
									<input id="recipe-adjustment-ratio" type="number" value={recipeRatio} oninput={(event) => adjustRecipeRatio(event.currentTarget.value)} step="0.1" min="0.1" class="w-full form-input-lg" />
								</label>
								<label for="recipe-adjustment-water">
									<span class="form-label">Water (g)</span>
									<input id="recipe-adjustment-water" type="number" value={waterAmount} oninput={(event) => adjustRecipeWater(event.currentTarget.value)} step="1" min="1" class="w-full form-input-lg" />
								</label>
							</div>
						</div>
					{/if}
				</div>
			{/if}
		</FormSection>

		<FormSection title="Coffee" description="Choose the beans and enter the dose and grind setting.">
			<div class="combo-select">
				<span class="form-label">Coffee bean <span class="text-red-500" aria-hidden="true">*</span></span>
				<EntityCombo
					entityType="bean"
					inputName="bean_rkey"
					apiEndpoint="/api/beans"
					suggestEndpoint="/api/suggestions/beans"
					placeholder="Search beans..."
					sectionLabel="Your beans"
					required={true}
					allowCreate={false}
					bind:rkey={beanRKey}
					bind:label={beanLabel}
					ariaLabel="Search coffee beans"
					onChange={(detail) => handleComboChange("bean", detail)}
				/>
			</div>
			{#if !activeRecipe}
				<Field label="Coffee amount (g)" helper="Amount of ground coffee used">
					<input type="number" bind:value={coffeeAmount} placeholder="e.g. 18" step="1" class="w-full form-input-lg" aria-invalid={coffeeAmountError} />
					{#if coffeeAmountError}<p class="text-xs text-red-600 mt-1">Coffee amount must be greater than 0.</p>{/if}
				</Field>
			{/if}
			<div class="combo-select">
				<span class="form-label">Grinder</span>
				<EntityCombo
					entityType="grinder"
					inputName="grinder_rkey"
					apiEndpoint="/api/grinders"
					suggestEndpoint="/api/suggestions/grinders"
					placeholder="Search grinders..."
					sectionLabel="Your grinders"
					bind:rkey={grinderRKey}
					bind:label={grinderLabel}
					ariaLabel="Search grinders"
					onChange={(detail) => handleComboChange("grinder", detail)}
				/>
			</div>
			<Field label="Grind size" helper={'Enter a grinder setting or description, such as "Medium" or "Fine"'}>
				<input type="text" bind:value={grindSize} placeholder="e.g. 18, Medium, 3.5, Fine" class="w-full form-input-lg" />
			</Field>
		</FormSection>

		<FormSection title="Brewing" description="Record the brewer, water, temperature, and time.">
			{#if showRecipeOverrides()}
				<div class="combo-select">
					<span class="form-label">Brew method</span>
					<EntityCombo
						entityType="brewer"
						inputName="brewer_rkey"
						apiEndpoint="/api/brewers"
						suggestEndpoint="/api/suggestions/brewers"
						placeholder="Search brew methods..."
						sectionLabel="Your brewers"
						bind:rkey={brewerRKey}
						bind:label={brewerLabel}
						ariaLabel="Search brew methods"
						onChange={(detail) => handleComboChange("brewer", detail)}
					/>
				</div>
				{#if !activeRecipe}
					<Field label="Water amount (g)" helper={pours.length > 0 ? "Total water (pours tracked separately below)" : "Total water used"}>
						<input type="number" bind:value={waterAmount} placeholder="e.g. 250" step="1" class="w-full form-input-lg" aria-invalid={waterAmountError} />
						{#if waterAmountError}<p class="text-xs text-red-600 mt-1">Water amount must be greater than 0.</p>{/if}
					</Field>
				{/if}
				<PoursEditor bind:pours expectedWater={waterAmount} />
			{/if}
			<Field label="Temperature (°F/°C)">
				<input type="number" bind:value={temperature} placeholder="e.g. 93.5" step="0.1" class="w-full form-input-lg" aria-invalid={temperatureError} />
				{#if temperatureError}<p class="text-xs text-red-600 mt-1">Temperature must be greater than 0.</p>{/if}
			</Field>
			<Field label="Brew time (s)">
				<input type="number" bind:value={timeSeconds} placeholder="e.g. 180" class="w-full form-input-lg" aria-invalid={timeSecondsError} />
				{#if timeSecondsError}<p class="text-xs text-red-600 mt-1">Brew time must be greater than 0.</p>{/if}
			</Field>
		</FormSection>

		{#if brewerCategory === "espresso"}
			<FormSection title="Espresso" description="Shot output, pressure, and pre-infusion.">
				<Field label="Yield weight (g)" helper="Weight of espresso output">
					<input type="number" bind:value={espressoYieldWeight} placeholder="e.g. 36" step="0.1" class="w-full form-input-lg" />
				</Field>
				<Field label="Pressure (bar)" helper="Brewing pressure">
					<input type="number" bind:value={espressoPressure} placeholder="e.g. 9" step="0.1" class="w-full form-input-lg" />
				</Field>
				<Field label="Pre-infusion time (s)">
					<input type="number" bind:value={espressoPreInfusionSeconds} placeholder="e.g. 5" class="w-full form-input-lg" />
				</Field>
			</FormSection>
		{/if}

		{#if brewerCategory === "pourover"}
			<FormSection title="Pour-over details" description="Record bloom, drawdown, bypass water, and filter details.">
				<div class="grid grid-cols-2 gap-4">
					<Field label="Bloom water (g)" helper="Water for bloom">
						<input type="number" bind:value={pouroverBloomWater} placeholder="e.g. 50" class="w-full form-input-lg" />
					</Field>
					<Field label="Bloom time (s)" helper="Bloom wait time">
						<input type="number" bind:value={pouroverBloomSeconds} placeholder="e.g. 45" class="w-full form-input-lg" />
					</Field>
				</div>
				<Field label="Drawdown time (s)" helper="Time after last pour until bed is dry">
					<input type="number" bind:value={pouroverDrawdownSeconds} placeholder="e.g. 30" class="w-full form-input-lg" />
				</Field>
				<Field label="Bypass water (g)" helper="Water added after brewing">
					<input type="number" bind:value={pouroverBypassWater} placeholder="e.g. 100" class="w-full form-input-lg" />
				</Field>
				<Field label="Filter" helper="Type of filter used">
					<input type="text" bind:value={pouroverFilter} placeholder="e.g. paper, metal, cloth" class="w-full form-input-lg" />
				</Field>
			</FormSection>
		{/if}

		<FormSection title="Results" description="Add tasting notes and an optional rating.">
			<Field label="Tasting notes">
				<textarea bind:value={tastingNotes} placeholder="Describe the flavors, aroma, and your thoughts..." rows="4" class="w-full form-input-lg"></textarea>
			</Field>
			<div>
				<label class="form-label" for="brew-rating">Rating</label>
				<input id="brew-rating" type="range" min="1" max="10" bind:value={rating} class="w-full accent-brown-700" />
				<div class="text-center text-2xl font-bold text-secondary">{rating}/10</div>
			</div>
		</FormSection>

		<div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-3 pt-2">
			<a href={isEdit && brew ? `/brews/${encodeURIComponent($session.did || $session.handle)}/${encodeURIComponent(brew.rkey)}` : "/my-coffee"} class="btn-secondary text-center">Cancel</a>
			<button type="submit" class="btn-primary" disabled={submitting}>
				{submitting ? "Saving..." : submitLabel}
			</button>
		</div>
	</form>

	{#snippet rail()}
		<RailSection title={beanLabel || "Untitled brew"} eyebrow="Brew session" lead={true}>
			{#if activeRecipe}
				<p>Recipe: {recipeSummary() || activeRecipe.name}</p>
			{:else}
				<p>No recipe selected.</p>
			{/if}
			{#if ratio !== null}
				<p>Ratio 1:{ratio.toFixed(1)} ({coffeeValue}g → {waterValue}g)</p>
			{:else if coffeeValue > 0 || waterValue > 0}
				<p>Add both coffee and water to see the brew ratio.</p>
			{/if}
			{#if brewerCategoryLabel}<p>Method: {brewerCategoryLabel}</p>{/if}
			{#if pourCount > 0}<p>{pourCount} {pourCount === 1 ? "pour" : "pours"} tracked.</p>{/if}
		</RailSection>
		<RailSection title="Brew details" eyebrow="Brew record">
			<p>The bean is required. Add any other measurements and equipment you want to reference later.</p>
		</RailSection>
		<RailSection title="Notes and recipe" eyebrow="Field notes">
			<p>Use tasting notes for flavors, aromas, and impressions. Put repeatable steps in the recipe.</p>
		</RailSection>
	{/snippet}
</FormWorkspace>

<style>
	.brew-form-sheet { padding-top: 1.5rem; }
	.brew-form-sheet :global(.form-section) { margin: 0; }
	.brew-form-sheet :global(.form-section) + :global(.form-section) { margin-top: 0; }
	.brew-form-sheet :global(.combo-select) { margin-bottom: 1.5rem; }
	.brew-form-sheet :global(.combo-select:last-child) { margin-bottom: 0; }
	.recipe-adjustment {
		margin-top: 0.75rem;
		padding: 1rem;
		border: 1px solid color-mix(in oklch, var(--type-recipe) 35%, var(--surface-border));
		border-radius: 0.5rem;
		background: color-mix(in oklch, var(--type-recipe-tint) 40%, var(--surface-bg));
	}
	.recipe-adjustment-heading { margin-bottom: 0.75rem; }
	.recipe-adjustment-heading :global(.text-helper) { margin-top: 0.25rem; }
	.recipe-adjustment-grid {
		display: grid;
		grid-template-columns: repeat(3, minmax(0, 1fr));
		gap: 0.75rem;
	}
	@media (max-width: 38rem) {
		.recipe-adjustment-grid { grid-template-columns: 1fr; }
	}
</style>
