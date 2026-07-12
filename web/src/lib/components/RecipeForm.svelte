<script lang="ts">
	import { goto } from "$app/navigation";
	import EntityCombo from "./EntityCombo.svelte";
	import PoursEditor from "./PoursEditor.svelte";
	import FormWorkspace from "./FormWorkspace.svelte";
	import LedgerHeader from "./LedgerHeader.svelte";
	import FormSection from "./FormSection.svelte";
	import RailSection from "./RailSection.svelte";
	import { APIError } from "$lib/api/client";
	import { createRecipe, updateRecipe } from "$lib/api/entities";
	import { appCache } from "$lib/stores/appCache";
	import { session } from "$lib/stores/session";
	import { pushToast } from "$lib/stores/toasts";
	import type { Recipe } from "$lib/types/entity_view";
	import type { Pour } from "./PoursEditor.svelte";

	const BREWER_TYPES = [
		{ value: "pourover", label: "Pour-over" },
		{ value: "espresso", label: "Espresso" },
		{ value: "immersion", label: "Immersion" },
		{ value: "mokapot", label: "Moka Pot" },
		{ value: "coldbrew", label: "Cold Brew" },
		{ value: "cupping", label: "Cupping" },
		{ value: "other", label: "Other" },
	];

	const BREWER_TYPE_LABELS: Record<string, string> = Object.fromEntries(
		BREWER_TYPES.map((t) => [t.value, t.label]),
	);

	type Props = {
		recipe: Recipe | null;
		isEdit: boolean;
	};

	let { recipe, isEdit }: Props = $props();

	// svelte-ignore state_referenced_locally
	let name = $state(recipe?.name ?? "");
	// svelte-ignore state_referenced_locally
	let brewerRKey = $state(recipe?.brewer_rkey ?? "");
	// svelte-ignore state_referenced_locally
	let brewerType = $state(recipe?.brewer_type ?? "");
	// svelte-ignore state_referenced_locally
	let coffeeAmount = $state<number | string>(recipe?.coffee_amount ?? "");
	// svelte-ignore state_referenced_locally
	let waterAmount = $state<number | string>(recipe?.water_amount ?? "");
	// svelte-ignore state_referenced_locally
	let notes = $state(recipe?.notes ?? "");
	// svelte-ignore state_referenced_locally
	let pours = $state<Pour[]>(
		(recipe?.pours ?? []).map((p) => ({
			water: p.water_amount ?? 0,
			time: p.time_seconds ?? 0,
		})),
	);

	let nameError = $state("");
	let formError = $state("");
	let submitting = $state(false);

	function validate(): boolean {
		nameError = name.trim() ? "" : "Name is required";
		return nameError === "";
	}

	function actor(): string {
		return $session.did || $session.handle;
	}

	function num(v: number | string): number {
		const n = Number(v);
		return Number.isFinite(n) ? n : 0;
	}

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (submitting) return;
		formError = "";
		if (!validate()) return;

		submitting = true;
		const input = {
			name: name.trim(),
			brewer_rkey: brewerRKey,
			brewer_type: brewerType,
			coffee_amount: num(coffeeAmount),
			water_amount: num(waterAmount),
			notes: notes.trim(),
			pours: pours
				.filter((p) => p.water !== "" || p.time !== "")
				.map((p) => ({ water_amount: num(p.water), time_seconds: num(p.time) })),
			...(recipe?.source_ref ? { source_ref: recipe.source_ref } : {}),
		};

		try {
			const saved = isEdit && recipe
				? await updateRecipe(fetch, recipe.rkey, input)
				: await createRecipe(fetch, input);
			appCache.invalidateCache();
			pushToast(isEdit ? "Recipe updated" : "Recipe added");
			const owner = actor();
			if (owner && saved.rkey) {
				await goto(`/recipes/${encodeURIComponent(owner)}/${encodeURIComponent(saved.rkey)}`);
				return;
			}
			await goto("/my-coffee");
		} catch (error) {
			if (error instanceof APIError) {
				nameError = error.fields?.name ?? nameError;
				formError = error.message;
			} else {
				formError = "Failed to save recipe";
			}
			pushToast(isEdit ? "Failed to update recipe" : "Failed to add recipe");
		} finally {
			submitting = false;
		}
	}

	// Live recipe context rail derivations.
	let coffeeValue = $derived(num(coffeeAmount));
	let waterValue = $derived(num(waterAmount));
	let ratio = $derived(
		coffeeValue > 0 && waterValue > 0 ? waterValue / coffeeValue : null,
	);
	let brewerTypeLabel = $derived(
		brewerType ? (BREWER_TYPE_LABELS[brewerType] ?? brewerType) : "",
	);
	let pourCount = $derived(
		pours.filter((p) => p.water !== "" || p.time !== "").length,
	);
	// Useful details for a reusable recipe: name, brewer, brewer type,
	// coffee, water, pours, notes.
	let completeness = $derived(
		[
			name.trim(),
			brewerRKey,
			brewerType,
			coffeeAmount !== "" ? String(coffeeAmount) : "",
			waterAmount !== "" ? String(waterAmount) : "",
			pourCount > 0 ? "pours" : "",
			notes.trim(),
		].filter(Boolean).length,
	);
</script>

<FormWorkspace>
	<LedgerHeader
		title={isEdit ? "Edit Recipe" : "Add a Recipe"}
		eyebrow="Recipe"
		description="Lay out a repeatable brew recipe so future sessions stay on track."
		showBack={true}
	/>

	<form class="recipe-form-sheet" novalidate onsubmit={submit}>
		{#if formError}
			<div class="alert-error" role="alert">{formError}</div>
		{/if}

		<FormSection title="Essentials">
			<div>
				<label class="form-label" for="recipe-name">Name <span class="text-red-500" aria-hidden="true">*</span></label>
				<p id="recipe-name-help" class="text-sm text-muted mb-2">A memorable name for this recipe.</p>
				<input
					id="recipe-name"
					name="name"
					type="text"
					class="w-full form-input-lg"
					bind:value={name}
					placeholder="e.g. V60 1:16 standard"
					aria-label="Name"
					required
					autocomplete="off"
					aria-invalid={nameError !== ""}
					aria-describedby={nameError ? "recipe-name-help recipe-name-error" : "recipe-name-help"}
					oninput={() => { if (nameError && name.trim()) nameError = ""; }}
				/>
				{#if nameError}
					<p id="recipe-name-error" class="text-sm text-red-600 mt-2" role="alert">{nameError}</p>
				{/if}
			</div>

			<div>
				<span class="form-label" id="recipe-brewer-label">Brewer</span>
				<p class="text-sm text-muted mb-2">Search your brewers or create a new one.</p>
				<EntityCombo
					entityType="brewer"
					apiEndpoint="/api/brewers"
					suggestEndpoint="/api/suggestions/brewers"
					inputName="brewer_rkey"
					placeholder="Search or create brewer"
					sectionLabel="Your brewers"
					bind:rkey={brewerRKey}
					ariaLabel="Brewer"
					allowCreate={true}
				/>
			</div>
		</FormSection>

		<FormSection title="Measurements" description="Coffee and water set the recipe's strength. Brewer type keeps it sorted.">
			<div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
				<div>
					<label class="form-label" for="recipe-brewer-type">Brewer type</label>
					<select id="recipe-brewer-type" name="brewer_type" class="w-full form-input-lg" bind:value={brewerType}>
						<option value="">Select type</option>
						{#each BREWER_TYPES as t}
							<option value={t.value}>{t.label}</option>
						{/each}
					</select>
				</div>
				<div>
					<label class="form-label" for="recipe-coffee">Coffee (g)</label>
					<input
						id="recipe-coffee"
						name="coffee_amount"
						type="number"
						step="0.1"
						min="0"
						class="w-full form-input-lg"
						bind:value={coffeeAmount}
						placeholder="15"
						aria-label="Coffee amount in grams"
					/>
				</div>
				<div>
					<label class="form-label" for="recipe-water">Water (g)</label>
					<input
						id="recipe-water"
						name="water_amount"
						type="number"
						step="0.1"
						min="0"
						class="w-full form-input-lg"
						bind:value={waterAmount}
						placeholder="240"
						aria-label="Water amount in grams"
					/>
				</div>
			</div>
		</FormSection>

		<FormSection title="Pours" description="Break the total water into stages for bloom and dilution.">
			<PoursEditor
				bind:pours
				expectedWater={waterAmount}
				title="Pours"
			/>
		</FormSection>

		<FormSection title="Recipe notes" description="Grind size, technique, and tasting targets belong here.">
			<div>
				<label class="form-label" for="recipe-notes">Notes</label>
				<textarea
					id="recipe-notes"
					name="notes"
					rows="4"
					class="w-full form-textarea"
					bind:value={notes}
					placeholder="Grind, technique, tasting notes, etc."
					aria-label="Notes"
				></textarea>
			</div>
		</FormSection>

		<div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-3 pt-2">
			<a href={isEdit && recipe ? `/recipes/${encodeURIComponent(actor())}/${encodeURIComponent(recipe.rkey)}` : "/my-coffee"} class="btn-secondary text-center">Cancel</a>
			<button type="submit" class="btn-primary" disabled={submitting}>
				{submitting ? "Saving..." : isEdit ? "Save Changes" : "Add Recipe"}
			</button>
		</div>
	</form>

	{#snippet rail()}
		<RailSection title={name.trim() || "Untitled recipe"} eyebrow="Recipe" lead={true}>
			<p>{brewerTypeLabel || "Brewer type not selected"}</p>
			{#if ratio !== null}
				<p>Ratio 1:{ratio.toFixed(1)} ({num(coffeeAmount)}g → {num(waterAmount)}g)</p>
			{:else}
				<p>Add coffee and water to see the brew ratio.</p>
			{/if}
		</RailSection>
		<RailSection title="Record completeness" eyebrow="Recipe status">
			<p>{completeness} of 7 useful details recorded.</p>
			<p>Name is required. Brewer, measurements, and pours make the recipe reproducible.</p>
		</RailSection>
		<RailSection title="What belongs here" eyebrow="Field notes">
			<p>Recipes are reusable templates. Use Notes for grind and technique; track actual results in a brew that references this recipe.</p>
		</RailSection>
	{/snippet}
</FormWorkspace>

<style>
	.recipe-form-sheet { padding-top: 1.5rem; }
	.recipe-form-sheet > :global(.alert-error) { margin-bottom: 1rem; }
	.recipe-form-sheet :global(.form-section) { margin: 0; }
</style>
