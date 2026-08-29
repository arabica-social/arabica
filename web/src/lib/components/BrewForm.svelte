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
	import { loadUserPrefs, userPrefs } from "../stores/userPrefs";
	import { session, warnIfSessionExpired } from "../stores/session";
	import { pushToast } from "../stores/toasts";
	import { goto } from "$app/navigation";
	import { createBrew, updateBrew, type BrewInput } from "../api/entities";
	import { computeAutofill, type AutofillField, type AutofillSuggestion } from "../utils/brewAutofill";
	import type { Brew, Brewer, Recipe } from "../types/entity_view";

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
	let historyBrews = $state<Brew[]>([]);
	let autofilled = $state<Partial<Record<AutofillField, AutofillSuggestion>>>({});

	// Guard against a stale recipe fetch from a previous selection writing
	// into the current form (see applyRecipe).
	let recipeGen = 0;

	function smartAutofillOn(): boolean {
		return !isEdit && !$userPrefs.loading && $userPrefs.smartAutofillEnabled;
	}

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

	function autofillBadge(field: AutofillField): string {
		const s = autofilled[field];
		if (!s) return "";
		const pct = s.total > 0 ? Math.round((s.matches / s.total) * 100) : 0;
		return `Autofilled · in ${s.matches} of ${s.total} brews (${pct}%)`;
	}

	function autofillInputClass(field: AutofillField): string {
		return `w-full form-input-lg${autofilled[field] ? " autofilled-input" : ""}`;
	}

	function clearFieldValue(field: AutofillField) {
		switch (field) {
			case "grinder_rkey": grinderRKey = ""; grinderLabel = ""; break;
			case "brewer_rkey":
				brewerRKey = "";
				brewerLabel = "";
				if (!activeRecipe) brewerCategory = "";
				break;
			case "method": method = ""; break;
			case "grind_size": grindSize = ""; break;
			case "temperature": temperature = ""; break;
			case "time_seconds": timeSeconds = ""; break;
			case "water_amount": waterAmount = ""; break;
			case "coffee_amount": coffeeAmount = ""; break;
			case "pourover_filter": pouroverFilter = ""; break;
			case "pourover_bloom_water": pouroverBloomWater = ""; break;
			case "pourover_bloom_seconds": pouroverBloomSeconds = ""; break;
			case "pourover_drawdown_seconds": pouroverDrawdownSeconds = ""; break;
			case "pourover_bypass_water": pouroverBypassWater = ""; break;
			case "espresso_yield_weight": espressoYieldWeight = ""; break;
			case "espresso_pressure": espressoPressure = ""; break;
			case "espresso_pre_infusion_seconds": espressoPreInfusionSeconds = ""; break;
		}
	}

	// Remove one field from autofill tracking because the user has edited it.
	// The value itself is left alone — the user's edit now owns it.
	function clearAutofillField(field: AutofillField) {
		if (!(field in autofilled)) return;
		const next = { ...autofilled };
		delete next[field];
		autofilled = next;
	}

	// Clear every still-tracked (not user-edited) autofilled field and drop the
	// badges. Used by the form-level Clear autofill action and when the
	// selection context changes before re-applying new suggestions.
	function clearAutofill() {
		for (const field of Object.keys(autofilled) as AutofillField[]) {
			clearFieldValue(field);
		}
		autofilled = {};
	}

	// Resolve the hydrated brewer type for an autofilled brewer rkey so the
	// Espresso/Pour-over sections become visible. Prefers the cached brewers
	// list; falls back to a brewer object from the user's brew history.
	function brewerTypeForRkey(rkey: string): string {
		if (!rkey) return "";
		const brewers = (cachedData.brewers as Brewer[] | undefined) ?? [];
		const cached = brewers.find((b) => b.rkey === rkey);
		if (cached?.brewer_type) return cached.brewer_type;
		for (const hb of historyBrews) {
			if (hb.brewer_rkey === rkey && hb.brewer_obj?.brewer_type) {
				return hb.brewer_obj.brewer_type;
			}
		}
		return "";
	}

	function applySuggestion(field: AutofillField, suggestion: AutofillSuggestion) {
		switch (field) {
			case "grinder_rkey":
				if (grinderRKey) return;
				grinderRKey = suggestion.rkey ?? suggestion.value;
				grinderLabel = suggestion.label ?? "";
				autofilled = { ...autofilled, grinder_rkey: suggestion };
				return;
			case "brewer_rkey":
				if (brewerRKey) return;
				brewerRKey = suggestion.rkey ?? suggestion.value;
				brewerLabel = suggestion.label ?? "";
				brewerCategory = normalizeBrewerCategory(brewerTypeForRkey(brewerRKey));
				autofilled = { ...autofilled, brewer_rkey: suggestion };
				return;
			case "method":
				if (method) return;
				method = suggestion.value;
				autofilled = { ...autofilled, method: suggestion };
				return;
			case "grind_size":
				if (grindSize) return;
				grindSize = suggestion.value;
				autofilled = { ...autofilled, grind_size: suggestion };
				return;
			case "temperature":
				if (temperature) return;
				temperature = suggestion.value;
				autofilled = { ...autofilled, temperature: suggestion };
				return;
			case "time_seconds":
				if (timeSeconds) return;
				timeSeconds = suggestion.value;
				autofilled = { ...autofilled, time_seconds: suggestion };
				return;
			case "water_amount":
				if (waterAmount) return;
				waterAmount = suggestion.value;
				autofilled = { ...autofilled, water_amount: suggestion };
				return;
			case "coffee_amount":
				if (coffeeAmount) return;
				coffeeAmount = suggestion.value;
				autofilled = { ...autofilled, coffee_amount: suggestion };
				return;
			case "pourover_filter":
				if (pouroverFilter) return;
				pouroverFilter = suggestion.value;
				autofilled = { ...autofilled, pourover_filter: suggestion };
				return;
			case "pourover_bloom_water":
				if (pouroverBloomWater) return;
				pouroverBloomWater = suggestion.value;
				autofilled = { ...autofilled, pourover_bloom_water: suggestion };
				return;
			case "pourover_bloom_seconds":
				if (pouroverBloomSeconds) return;
				pouroverBloomSeconds = suggestion.value;
				autofilled = { ...autofilled, pourover_bloom_seconds: suggestion };
				return;
			case "pourover_drawdown_seconds":
				if (pouroverDrawdownSeconds) return;
				pouroverDrawdownSeconds = suggestion.value;
				autofilled = { ...autofilled, pourover_drawdown_seconds: suggestion };
				return;
			case "pourover_bypass_water":
				if (pouroverBypassWater) return;
				pouroverBypassWater = suggestion.value;
				autofilled = { ...autofilled, pourover_bypass_water: suggestion };
				return;
			case "espresso_yield_weight":
				if (espressoYieldWeight) return;
				espressoYieldWeight = suggestion.value;
				autofilled = { ...autofilled, espresso_yield_weight: suggestion };
				return;
			case "espresso_pressure":
				if (espressoPressure) return;
				espressoPressure = suggestion.value;
				autofilled = { ...autofilled, espresso_pressure: suggestion };
				return;
			case "espresso_pre_infusion_seconds":
				if (espressoPreInfusionSeconds) return;
				espressoPreInfusionSeconds = suggestion.value;
				autofilled = { ...autofilled, espresso_pre_infusion_seconds: suggestion };
				return;
		}
	}

	// Recompute suggestions for the current bean + recipe selection and apply
	// them to still-empty fields. This must run only for new brews, only when
	// the preference is enabled, only with a bean selected, and only after a
	// recipe (if any) has populated its fields.
	function applyAutofill() {
		if (!smartAutofillOn()) {
			clearAutofill();
			return;
		}
		if (!beanRKey) {
			autofilled = {};
			return;
		}
		const result = computeAutofill(historyBrews, {
			beanRKey,
			recipeRKey: recipeRKeyValue || undefined,
			recipeOwnerDID: recipeOwner || undefined,
		});
		if (!result) {
			autofilled = {};
			return;
		}
		for (const [field, suggestion] of Object.entries(result.fields) as [AutofillField, AutofillSuggestion][]) {
			applySuggestion(field, suggestion);
		}
	}

	let autofillCount = $derived(Object.keys(autofilled).length);

	$effect(() => {
		const enabled = $userPrefs.smartAutofillEnabled;
		const loading = $userPrefs.loading;
		if (!loading && !enabled && autofillCount > 0) clearAutofill();
	});

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
			// Own recipes selected from the cached list may not carry author_did;
			// use the authenticated DID so recipe-pair matching remains owner-safe.
			recipeOwner = (detail.owner as string) || (entity?.author_did as string) || $session.did;
			// The recipe owns dose/water/brewer, so drop any previous autofill,
			// let applyRecipe populate those fields, then autofill the leftovers.
			clearAutofill();
			void applyRecipe(rkey, recipeOwner).then(() => applyAutofill());
			return;
		}
		if (type === "bean") {
			beanRKey = rkey;
			clearAutofill();
			applyAutofill();
			return;
		}
		if (type === "grinder") {
			grinderRKey = rkey;
			clearAutofillField("grinder_rkey");
			return;
		}
		if (type === "brewer") {
			brewerRKey = rkey;
			brewerCategory = normalizeBrewerCategory((entity?.brewer_type as string) ?? "");
			clearAutofillField("brewer_rkey");
		}
	}

	function clearCombo(type: ComboType) {
		if (type === "recipe") { recipeRKeyValue = ""; recipeLabel = ""; activeRecipe = null; recipeOwner = ""; recipeExpanded = false; recipeRatio = ""; recipePours = []; clearAutofill(); applyAutofill(); }
		if (type === "bean") { beanRKey = ""; beanLabel = ""; clearAutofill(); }
		if (type === "grinder") { grinderRKey = ""; grinderLabel = ""; clearAutofillField("grinder_rkey"); }
		if (type === "brewer") { brewerRKey = ""; brewerLabel = ""; brewerCategory = ""; clearAutofillField("brewer_rkey"); }
	}

	async function applyRecipe(selectedRKey: string, selectedOwner = "") {
		if (!selectedRKey) { activeRecipe = null; recipeOwner = ""; recipeExpanded = false; return; }
		const gen = ++recipeGen;
		// The endpoint defaults to the authenticated user's recipe when no
		// owner query is supplied. Still retain the DID for owner-safe matching.
		recipeOwner = selectedOwner || $session.did;
		const ownerQuery = selectedOwner ? `?owner=${encodeURIComponent(selectedOwner)}` : "";
		try {
			const res = await fetch(`/api/recipes/${selectedRKey}${ownerQuery}`, { credentials: "same-origin" });
			if (!res.ok) return;
			const recipe = (await res.json()) as Recipe;
			if (gen !== recipeGen) return; // A newer recipe selection superseded this one.
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
		// Load the app cache and preference together. This prevents a fast bean
		// selection from applying autofill before a saved "off" preference has
		// finished loading.
		const preferences = isEdit ? Promise.resolve() : loadUserPrefs();
		void Promise.all([appCache.getData(), preferences]).then(([data]) => {
			if (data) {
				cachedData = data;
				historyBrews = (data.brews as Brew[]) ?? [];
				if (beanRKey) applyAutofill();
			}
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
		{#if autofillCount > 0}
			<div class="autofill-summary">
				<p class="text-helper">Some empty fields were filled from your brew history.</p>
				<button type="button" onclick={clearAutofill} class="btn-secondary text-sm">Clear autofill</button>
			</div>
		{/if}
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
				<Field label="Coffee amount (g)" helper="Amount of ground coffee used" badge={autofillBadge("coffee_amount")}>
					<input type="number" bind:value={coffeeAmount} oninput={() => clearAutofillField("coffee_amount")} placeholder="e.g. 18" step="1" class={autofillInputClass("coffee_amount")} aria-invalid={coffeeAmountError} />
					{#if coffeeAmountError}<p class="text-xs text-red-600 mt-1">Coffee amount must be greater than 0.</p>{/if}
				</Field>
			{/if}
			<div class="combo-select">
				<span class="form-label">Grinder</span>
				<EntityCombo
					entityType="grinder"
					inputName="grinder_rkey"
					inputClass={autofillInputClass("grinder_rkey")}
					apiEndpoint="/api/grinders"
					suggestEndpoint="/api/suggestions/grinders"
					placeholder="Search grinders..."
					sectionLabel="Your grinders"
					bind:rkey={grinderRKey}
					bind:label={grinderLabel}
					ariaLabel="Search grinders"
					onChange={(detail) => handleComboChange("grinder", detail)}
				/>
				<div class="autofill-status" aria-live="polite">
					{#if autofillBadge("grinder_rkey")}<span class="autofill-badge">{autofillBadge("grinder_rkey")}</span>{/if}
				</div>
			</div>
			<Field label="Grind size" helper={'Enter a grinder setting or description, such as "Medium" or "Fine"'} badge={autofillBadge("grind_size")}>
				<input type="text" bind:value={grindSize} oninput={() => clearAutofillField("grind_size")} placeholder="e.g. 18, Medium, 3.5, Fine" class={autofillInputClass("grind_size")} />
			</Field>
		</FormSection>

		<FormSection title="Brewing" description="Record the brewer, water, temperature, and time.">
			{#if showRecipeOverrides()}
				<div class="combo-select">
					<span class="form-label">Brew method</span>
					<EntityCombo
						entityType="brewer"
						inputName="brewer_rkey"
						inputClass={autofillInputClass("brewer_rkey")}
						apiEndpoint="/api/brewers"
						suggestEndpoint="/api/suggestions/brewers"
						placeholder="Search brew methods..."
						sectionLabel="Your brewers"
						bind:rkey={brewerRKey}
						bind:label={brewerLabel}
						ariaLabel="Search brew methods"
						onChange={(detail) => handleComboChange("brewer", detail)}
					/>
					<div class="autofill-status" aria-live="polite">
						{#if autofillBadge("brewer_rkey")}<span class="autofill-badge">{autofillBadge("brewer_rkey")}</span>{/if}
					</div>
				</div>
				{#if !activeRecipe}
					<Field label="Water amount (g)" helper={pours.length > 0 ? "Total water (pours tracked separately below)" : "Total water used"} badge={autofillBadge("water_amount")}>
						<input type="number" bind:value={waterAmount} oninput={() => clearAutofillField("water_amount")} placeholder="e.g. 250" step="1" class={autofillInputClass("water_amount")} aria-invalid={waterAmountError} />
						{#if waterAmountError}<p class="text-xs text-red-600 mt-1">Water amount must be greater than 0.</p>{/if}
					</Field>
				{/if}
				<PoursEditor bind:pours expectedWater={waterAmount} />
			{/if}
			<Field label="Temperature (°F/°C)" badge={autofillBadge("temperature")}>
				<input type="number" bind:value={temperature} oninput={() => clearAutofillField("temperature")} placeholder="e.g. 93.5" step="0.1" class={autofillInputClass("temperature")} aria-invalid={temperatureError} />
				{#if temperatureError}<p class="text-xs text-red-600 mt-1">Temperature must be greater than 0.</p>{/if}
			</Field>
			<Field label="Brew time (s)" badge={autofillBadge("time_seconds")}>
				<input type="number" bind:value={timeSeconds} oninput={() => clearAutofillField("time_seconds")} placeholder="e.g. 180" class={autofillInputClass("time_seconds")} aria-invalid={timeSecondsError} />
				{#if timeSecondsError}<p class="text-xs text-red-600 mt-1">Brew time must be greater than 0.</p>{/if}
			</Field>
		</FormSection>

		{#if brewerCategory === "espresso"}
			<FormSection title="Espresso" description="Shot output, pressure, and pre-infusion.">
				<Field label="Yield weight (g)" helper="Weight of espresso output" badge={autofillBadge("espresso_yield_weight")}>
					<input type="number" bind:value={espressoYieldWeight} oninput={() => clearAutofillField("espresso_yield_weight")} placeholder="e.g. 36" step="0.1" class={autofillInputClass("espresso_yield_weight")} />
				</Field>
				<Field label="Pressure (bar)" helper="Brewing pressure" badge={autofillBadge("espresso_pressure")}>
					<input type="number" bind:value={espressoPressure} oninput={() => clearAutofillField("espresso_pressure")} placeholder="e.g. 9" step="0.1" class={autofillInputClass("espresso_pressure")} />
				</Field>
				<Field label="Pre-infusion time (s)" badge={autofillBadge("espresso_pre_infusion_seconds")}>
					<input type="number" bind:value={espressoPreInfusionSeconds} oninput={() => clearAutofillField("espresso_pre_infusion_seconds")} placeholder="e.g. 5" class={autofillInputClass("espresso_pre_infusion_seconds")} />
				</Field>
			</FormSection>
		{/if}

		{#if brewerCategory === "pourover"}
			<FormSection title="Pour-over details" description="Record bloom, drawdown, bypass water, and filter details.">
				<div class="grid grid-cols-2 gap-4">
					<Field label="Bloom water (g)" helper="Water for bloom" badge={autofillBadge("pourover_bloom_water")}>
						<input type="number" bind:value={pouroverBloomWater} oninput={() => clearAutofillField("pourover_bloom_water")} placeholder="e.g. 50" class={autofillInputClass("pourover_bloom_water")} />
					</Field>
					<Field label="Bloom time (s)" helper="Bloom wait time" badge={autofillBadge("pourover_bloom_seconds")}>
						<input type="number" bind:value={pouroverBloomSeconds} oninput={() => clearAutofillField("pourover_bloom_seconds")} placeholder="e.g. 45" class={autofillInputClass("pourover_bloom_seconds")} />
					</Field>
				</div>
				<Field label="Drawdown time (s)" helper="Time after last pour until bed is dry" badge={autofillBadge("pourover_drawdown_seconds")}>
					<input type="number" bind:value={pouroverDrawdownSeconds} oninput={() => clearAutofillField("pourover_drawdown_seconds")} placeholder="e.g. 30" class={autofillInputClass("pourover_drawdown_seconds")} />
				</Field>
				<Field label="Bypass water (g)" helper="Water added after brewing" badge={autofillBadge("pourover_bypass_water")}>
					<input type="number" bind:value={pouroverBypassWater} oninput={() => clearAutofillField("pourover_bypass_water")} placeholder="e.g. 100" class={autofillInputClass("pourover_bypass_water")} />
				</Field>
				<Field label="Filter" helper="Type of filter used" badge={autofillBadge("pourover_filter")}>
					<input type="text" bind:value={pouroverFilter} oninput={() => clearAutofillField("pourover_filter")} placeholder="e.g. paper, metal, cloth" class={autofillInputClass("pourover_filter")} />
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
	.brew-form-sheet .autofill-summary {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		margin-bottom: 1.25rem;
		padding: 0.65rem 0.9rem;
		border: 1px dashed var(--card-border);
		border-radius: 0.5rem;
		background: color-mix(in oklch, var(--surface-bg) 74%, transparent);
	}
	.brew-form-sheet :global(.autofilled-input) {
		color: var(--autofill-text);
	}
	.brew-form-sheet :global(.autofill-status) {
		display: flex;
		align-items: center;
		min-height: 1.25rem;
		margin-top: 0.25rem;
	}
	.brew-form-sheet :global(.autofill-badge) {
		display: inline-flex;
		align-items: center;
		min-height: 1.1rem;
		padding: 0.15rem 0.5rem;
		border: 1px dashed var(--autofill-text);
		border-radius: 999px;
		background: color-mix(in oklch, var(--surface-bg) 74%, transparent);
		color: var(--autofill-text);
		font-size: 0.68rem;
		font-weight: 600;
		letter-spacing: 0.02em;
		line-height: 1.1;
	}
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
