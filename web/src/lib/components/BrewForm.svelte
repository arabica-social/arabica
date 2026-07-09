<script lang="ts">
	import { onMount } from "svelte";
	import EntityCombo from "./EntityCombo.svelte";
	import Field from "./BrewFormField.svelte";
	import PoursEditor from "./PoursEditor.svelte";
	import BackButton from "./BackButton.svelte";
	import { appCache } from "../stores/appCache";
	import { pushToast } from "../stores/toasts";
	import { goto } from "$app/navigation";
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
	let brewerCategory = $state("");

	// Form state
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
		if (type === "recipe") { recipeRKeyValue = ""; recipeLabel = ""; activeRecipe = null; recipeOwner = ""; recipeExpanded = false; }
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
			pours = (recipe.pours ?? []).map((p) => ({ water: String(p.water_amount ?? ""), time: String(p.time_seconds ?? "") }));
			const recipeBrewerType = recipe.brewer_type || recipe.brewer_obj?.brewer_type || "";
			if (recipeBrewerType) brewerCategory = normalizeBrewerCategory(recipeBrewerType);
		} catch {
			// Ignore — user can fill manually.
		}
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
	});

	async function submitForm(e: SubmitEvent) {
		e.preventDefault();
		if (submitting) return;
		submitting = true;

		const formData = new FormData();
		formData.set("bean_rkey", beanRKey);
		if (grinderRKey) formData.set("grinder_rkey", grinderRKey);
		if (brewerRKey) formData.set("brewer_rkey", brewerRKey);
		if (recipeRKeyValue) {
			formData.set("recipe_rkey", recipeRKeyValue);
			if (recipeOwner) formData.set("recipe_owner_did", recipeOwner);
		}
		if (method) formData.set("method", method);
		if (coffeeAmount) formData.set("coffee_amount", coffeeAmount);
		if (waterAmount) formData.set("water_amount", waterAmount);
		if (grindSize) formData.set("grind_size", grindSize);
		if (temperature) formData.set("temperature", temperature);
		if (timeSeconds) formData.set("time_seconds", timeSeconds);
		if (tastingNotes) formData.set("tasting_notes", tastingNotes);
		if (rating) formData.set("rating", rating);
		// Pours
		pours.forEach((pour, i) => {
			if (pour.water) formData.set(`pour_water_${i}`, pour.water);
			if (pour.time) formData.set(`pour_time_${i}`, pour.time);
		});
		// Espresso params
		if (espressoYieldWeight) formData.set("espresso_yield_weight", espressoYieldWeight);
		if (espressoPressure) formData.set("espresso_pressure", espressoPressure);
		if (espressoPreInfusionSeconds) formData.set("espresso_pre_infusion_seconds", espressoPreInfusionSeconds);
		// Pourover params
		if (pouroverBloomWater) formData.set("pourover_bloom_water", pouroverBloomWater);
		if (pouroverBloomSeconds) formData.set("pourover_bloom_seconds", pouroverBloomSeconds);
		if (pouroverDrawdownSeconds) formData.set("pourover_drawdown_seconds", pouroverDrawdownSeconds);
		if (pouroverBypassWater) formData.set("pourover_bypass_water", pouroverBypassWater);
		if (pouroverFilter) formData.set("pourover_filter", pouroverFilter);

		const url = isEdit ? `/brews/${brew?.rkey ?? ""}` : "/brews";
		const httpMethod = isEdit ? "PUT" : "POST";

		try {
			const res = await fetch(url, {
				method: httpMethod,
				credentials: "same-origin",
				headers: { Accept: "application/json" },
				body: formData,
			});
			if (!res.ok) {
				const text = await res.text().catch(() => "");
				throw new Error(`Save failed: ${res.status} ${text}`);
			}
			const data = await res.json();
			const createdBrew = data.brew ?? data;
			pushToast(isEdit ? "Brew updated!" : "Brew saved!");
			// Redirect to the brew view or my-coffee.
			const actor = createdBrew.author_did ?? createdBrew.author?.did ?? "";
			const rkey = createdBrew.rkey ?? "";
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

	let submitLabel = $derived(isEdit ? "Update Brew" : "Save Brew");
</script>

<div class="page-container-sm">
	<div class="card card-inner">
		<div class="flex items-center gap-3 mb-6">
			<BackButton />
			<h2 class="text-2xl font-semibold text-primary">{isEdit ? "Edit Brew" : "New Brew"}</h2>
		</div>

		<form onsubmit={submitForm} class="space-y-6">
			<!-- Recipe (optional) -->
			<div class="combo-select">
				<span class="form-label">Recipe (Optional)</span>
				<p class="text-sm text-muted mb-2">Select a recipe to autofill brew parameters</p>
				<div class="alert-warning px-3 py-2 mb-2 text-xs">
					Recipes are in early alpha, the format may change. Your brew data won't be affected.
				</div>
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
						<button type="button" onclick={() => (recipeExpanded = !recipeExpanded)} class="text-sm btn-secondary">
							{recipeExpanded ? "Collapse" : "Edit"}
						</button>
					</div>
				</div>
			{/if}

			<!-- Coffee section -->
			<fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
				<legend class="text-sm font-semibold text-secondary px-2">Coffee</legend>
				<div class="combo-select">
					<span class="form-label">Coffee Bean <span class="text-red-500">*</span></span>
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
				{#if showRecipeOverrides()}
					<Field label="Coffee Amount (grams)" helper="Amount of ground coffee used">
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
				<Field label="Grind Size" helper={'Enter a number (grinder setting) or description (e.g. "Medium", "Fine")'}>
					<input type="text" bind:value={grindSize} placeholder="e.g. 18, Medium, 3.5, Fine" class="w-full form-input-lg" />
				</Field>
			</fieldset>

			<!-- Brewing section -->
			<fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
				<legend class="text-sm font-semibold text-secondary px-2">Brewing</legend>
				{#if showRecipeOverrides()}
					<div class="combo-select">
						<span class="form-label">Brew Method</span>
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
					<Field label="Water Amount (grams)" helper={pours.length > 0 ? "Total water (pours tracked separately below)" : "Total water used"}>
						<input type="number" bind:value={waterAmount} placeholder="e.g. 250" step="1" class="w-full form-input-lg" aria-invalid={waterAmountError} />
						{#if waterAmountError}<p class="text-xs text-red-600 mt-1">Water amount must be greater than 0.</p>{/if}
					</Field>
					<PoursEditor bind:pours expectedWater={waterAmount} />
				{/if}
				<Field label="Temperature (°F/°C)">
					<input type="number" bind:value={temperature} placeholder="e.g. 93.5" step="0.1" class="w-full form-input-lg" aria-invalid={temperatureError} />
					{#if temperatureError}<p class="text-xs text-red-600 mt-1">Temperature must be greater than 0.</p>{/if}
				</Field>
				<Field label="Brew Time (seconds)">
					<input type="number" bind:value={timeSeconds} placeholder="e.g. 180" class="w-full form-input-lg" aria-invalid={timeSecondsError} />
					{#if timeSecondsError}<p class="text-xs text-red-600 mt-1">Brew time must be greater than 0.</p>{/if}
				</Field>
			</fieldset>

			<!-- Espresso params -->
			{#if brewerCategory === "espresso"}
				<fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
					<legend class="text-sm font-semibold text-secondary px-2">Espresso</legend>
					<Field label="Yield Weight (grams)" helper="Weight of espresso output">
						<input type="number" bind:value={espressoYieldWeight} placeholder="e.g. 36" step="0.1" class="w-full form-input-lg" />
					</Field>
					<Field label="Pressure (bar)" helper="Brewing pressure">
						<input type="number" bind:value={espressoPressure} placeholder="e.g. 9" step="0.1" class="w-full form-input-lg" />
					</Field>
					<Field label="Pre-infusion Time (seconds)">
						<input type="number" bind:value={espressoPreInfusionSeconds} placeholder="e.g. 5" class="w-full form-input-lg" />
					</Field>
				</fieldset>
			{/if}

			<!-- Pourover params -->
			{#if brewerCategory === "pourover"}
				<fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
					<legend class="text-sm font-semibold text-secondary px-2">Pour-over Details</legend>
					<div class="grid grid-cols-2 gap-4">
						<Field label="Bloom Water (grams)" helper="Water for bloom">
							<input type="number" bind:value={pouroverBloomWater} placeholder="e.g. 50" class="w-full form-input-lg" />
						</Field>
						<Field label="Bloom Time (seconds)" helper="Bloom wait time">
							<input type="number" bind:value={pouroverBloomSeconds} placeholder="e.g. 45" class="w-full form-input-lg" />
						</Field>
					</div>
					<Field label="Drawdown Time (seconds)" helper="Time after last pour until bed is dry">
						<input type="number" bind:value={pouroverDrawdownSeconds} placeholder="e.g. 30" class="w-full form-input-lg" />
					</Field>
					<Field label="Bypass Water (grams)" helper="Water added after brewing">
						<input type="number" bind:value={pouroverBypassWater} placeholder="e.g. 100" class="w-full form-input-lg" />
					</Field>
					<Field label="Filter" helper="Type of filter used">
						<input type="text" bind:value={pouroverFilter} placeholder="e.g. paper, metal, cloth" class="w-full form-input-lg" />
					</Field>
				</fieldset>
			{/if}

			<!-- Results -->
			<fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
				<legend class="text-sm font-semibold text-secondary px-2">Results</legend>
				<Field label="Tasting Notes">
					<textarea bind:value={tastingNotes} placeholder="Describe the flavors, aroma, and your thoughts..." rows="4" class="w-full form-input-lg"></textarea>
				</Field>
				<div>
					<label class="form-label" for="brew-rating">Rating</label>
					<input id="brew-rating" type="range" min="1" max="10" bind:value={rating} class="w-full accent-brown-700" />
					<div class="text-center text-2xl font-bold text-secondary">{rating}/10</div>
				</div>
			</fieldset>

			<button type="submit" class="w-full btn-primary py-3 px-6 rounded-xl font-semibold text-lg shadow-lg hover:shadow-xl" disabled={submitting}>
				{submitting ? "Saving..." : submitLabel}
			</button>
		</form>
	</div>
</div>
