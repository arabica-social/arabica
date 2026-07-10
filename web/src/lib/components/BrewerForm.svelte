<script lang="ts">
	import { goto } from "$app/navigation";
	import BackButton from "./BackButton.svelte";
	import { APIError } from "$lib/api/client";
	import { createBrewer, updateBrewer } from "$lib/api/entities";
	import { appCache } from "$lib/stores/appCache";
	import { session } from "$lib/stores/session";
	import { pushToast } from "$lib/stores/toasts";
	import type { Brewer } from "$lib/types/entity_view";

	// Canonical brewer type values (internal/arabica/entities/models.go).
	const BREWER_TYPES = [
		{ value: "pourover", label: "Pour-over" },
		{ value: "espresso", label: "Espresso" },
		{ value: "immersion", label: "Immersion" },
		{ value: "mokapot", label: "Moka Pot" },
		{ value: "coldbrew", label: "Cold Brew" },
		{ value: "cupping", label: "Cupping" },
		{ value: "other", label: "Other" },
	];

	type Props = {
		brewer: Brewer | null;
		isEdit: boolean;
	};

	let { brewer, isEdit }: Props = $props();

	// svelte-ignore state_referenced_locally
	let name = $state(brewer?.name ?? "");
	// svelte-ignore state_referenced_locally
	let brewerType = $state(brewer?.brewer_type ?? "");
	// svelte-ignore state_referenced_locally
	let description = $state(brewer?.description ?? "");
	// svelte-ignore state_referenced_locally
	let link = $state(brewer?.link ?? "");

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

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (submitting) return;
		formError = "";
		if (!validate()) return;

		submitting = true;
		const input = {
			name: name.trim(),
			brewer_type: brewerType,
			description: description.trim(),
			link: link.trim(),
		};

		try {
			const saved = isEdit && brewer
				? await updateBrewer(fetch, brewer.rkey, input)
				: await createBrewer(fetch, input);
			appCache.invalidateCache();
			pushToast(isEdit ? "Brewer updated" : "Brewer added");
			const owner = actor();
			if (owner && saved.rkey) {
				await goto(`/brewers/${encodeURIComponent(owner)}/${encodeURIComponent(saved.rkey)}`);
				return;
			}
			await goto("/my-coffee");
		} catch (error) {
			if (error instanceof APIError) {
				nameError = error.fields?.name ?? nameError;
				formError = error.message;
			} else {
				formError = "Failed to save brewer";
			}
			pushToast(isEdit ? "Failed to update brewer" : "Failed to add brewer");
		} finally {
			submitting = false;
		}
	}
</script>

<div class="page-container-sm">
	<div class="card card-inner">
		<div class="flex items-center gap-3 mb-6">
			<BackButton />
			<div>
				<p class="text-xs font-semibold uppercase tracking-wider text-faint">Brewer</p>
				<h1 class="text-2xl font-semibold text-primary">{isEdit ? "Edit Brewer" : "Add a Brewer"}</h1>
			</div>
		</div>

		<form class="space-y-6" novalidate onsubmit={submit}>
			{#if formError}
				<div class="alert-error" role="alert">{formError}</div>
			{/if}

			<div>
				<label class="form-label" for="brewer-name">Name <span class="text-red-500" aria-hidden="true">*</span></label>
				<p id="brewer-name-help" class="text-sm text-muted mb-2">The brewer name shown in your journal.</p>
				<input
					id="brewer-name"
					name="name"
					type="text"
					class="w-full form-input-lg"
					bind:value={name}
					placeholder="e.g. Hario V60-02"
					aria-label="Name"
					required
					autocomplete="off"
					aria-invalid={nameError !== ""}
					aria-describedby={nameError ? "brewer-name-help brewer-name-error" : "brewer-name-help"}
					oninput={() => { if (nameError && name.trim()) nameError = ""; }}
				/>
				{#if nameError}
					<p id="brewer-name-error" class="text-sm text-red-600 mt-2" role="alert">{nameError}</p>
				{/if}
			</div>

			<div>
				<label class="form-label" for="brewer-type">Type</label>
				<select id="brewer-type" name="brewer_type" class="w-full form-input-lg" bind:value={brewerType}>
					<option value="">Select type</option>
					{#each BREWER_TYPES as t}
						<option value={t.value}>{t.label}</option>
					{/each}
				</select>
			</div>

			<div>
				<label class="form-label" for="brewer-description">Description</label>
				<p id="brewer-description-help" class="text-sm text-muted mb-2">Notes about this brewer.</p>
				<textarea
					id="brewer-description"
					name="description"
					rows="3"
					class="w-full form-textarea"
					bind:value={description}
					placeholder="Capacity, material, etc."
					aria-label="Description"
					aria-describedby="brewer-description-help"
				></textarea>
			</div>

			<div>
				<label class="form-label" for="brewer-link">Link</label>
				<input
					id="brewer-link"
					name="link"
					type="url"
					class="w-full form-input-lg"
					bind:value={link}
					placeholder="https://example.com/brewer"
					aria-label="Link"
					autocomplete="url"
				/>
			</div>

			<div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-3 pt-2">
				<a href={isEdit && brewer ? `/brewers/${encodeURIComponent(actor())}/${encodeURIComponent(brewer.rkey)}` : "/my-coffee"} class="btn-secondary text-center">Cancel</a>
				<button type="submit" class="btn-primary" disabled={submitting}>
					{submitting ? "Saving..." : isEdit ? "Save Changes" : "Add Brewer"}
				</button>
			</div>
		</form>
	</div>
</div>
