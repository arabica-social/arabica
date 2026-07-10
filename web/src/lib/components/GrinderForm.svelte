<script lang="ts">
	import { goto } from "$app/navigation";
	import BackButton from "./BackButton.svelte";
	import { APIError } from "$lib/api/client";
	import { createGrinder, updateGrinder } from "$lib/api/entities";
	import { appCache } from "$lib/stores/appCache";
	import { session } from "$lib/stores/session";
	import { pushToast } from "$lib/stores/toasts";
	import type { Grinder } from "$lib/types/entity_view";

	const GRINDER_TYPES = ["Hand", "Electric", "Portable Electric"];
	const BURR_TYPES = ["Conical", "Flat"];

	type Props = {
		grinder: Grinder | null;
		isEdit: boolean;
	};

	let { grinder, isEdit }: Props = $props();

	// svelte-ignore state_referenced_locally
	let name = $state(grinder?.name ?? "");
	// svelte-ignore state_referenced_locally
	let grinderType = $state(grinder?.grinder_type ?? "");
	// svelte-ignore state_referenced_locally
	let burrType = $state(grinder?.burr_type ?? "");
	// svelte-ignore state_referenced_locally
	let notes = $state(grinder?.notes ?? "");
	// svelte-ignore state_referenced_locally
	let link = $state(grinder?.link ?? "");

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
			grinder_type: grinderType,
			burr_type: burrType,
			notes: notes.trim(),
			link: link.trim(),
			...(grinder?.source_ref ? { source_ref: grinder.source_ref } : {}),
		};

		try {
			const saved = isEdit && grinder
				? await updateGrinder(fetch, grinder.rkey, input)
				: await createGrinder(fetch, input);
			appCache.invalidateCache();
			pushToast(isEdit ? "Grinder updated" : "Grinder added");
			const owner = actor();
			if (owner && saved.rkey) {
				await goto(`/grinders/${encodeURIComponent(owner)}/${encodeURIComponent(saved.rkey)}`);
				return;
			}
			await goto("/my-coffee");
		} catch (error) {
			if (error instanceof APIError) {
				nameError = error.fields?.name ?? nameError;
				formError = error.message;
			} else {
				formError = "Failed to save grinder";
			}
			pushToast(isEdit ? "Failed to update grinder" : "Failed to add grinder");
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
				<p class="text-xs font-semibold uppercase tracking-wider text-faint">Grinder</p>
				<h1 class="text-2xl font-semibold text-primary">{isEdit ? "Edit Grinder" : "Add a Grinder"}</h1>
			</div>
		</div>

		<form class="space-y-6" novalidate onsubmit={submit}>
			{#if formError}
				<div class="alert-error" role="alert">{formError}</div>
			{/if}

			<div>
				<label class="form-label" for="grinder-name">Name <span class="text-red-500" aria-hidden="true">*</span></label>
				<p id="grinder-name-help" class="text-sm text-muted mb-2">The grinder make/model shown in your journal.</p>
				<input
					id="grinder-name"
					name="name"
					type="text"
					class="w-full form-input-lg"
					bind:value={name}
					placeholder="e.g. Comandante C40"
					aria-label="Name"
					required
					autocomplete="off"
					aria-invalid={nameError !== ""}
					aria-describedby={nameError ? "grinder-name-help grinder-name-error" : "grinder-name-help"}
					oninput={() => { if (nameError && name.trim()) nameError = ""; }}
				/>
				{#if nameError}
					<p id="grinder-name-error" class="text-sm text-red-600 mt-2" role="alert">{nameError}</p>
				{/if}
			</div>

			<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
				<div>
					<label class="form-label" for="grinder-type">Type</label>
					<select id="grinder-type" name="grinder_type" class="w-full form-input-lg" bind:value={grinderType}>
						<option value="">Select type</option>
						{#each GRINDER_TYPES as t}
							<option value={t}>{t}</option>
						{/each}
					</select>
				</div>
				<div>
					<label class="form-label" for="grinder-burr-type">Burr type</label>
					<select id="grinder-burr-type" name="burr_type" class="w-full form-input-lg" bind:value={burrType}>
						<option value="">Select burr type</option>
						{#each BURR_TYPES as t}
							<option value={t}>{t}</option>
						{/each}
					</select>
				</div>
			</div>

			<div>
				<label class="form-label" for="grinder-notes">Notes</label>
				<p id="grinder-notes-help" class="text-sm text-muted mb-2">Personal notes about this grinder.</p>
				<textarea
					id="grinder-notes"
					name="notes"
					rows="3"
					class="w-full form-textarea"
					bind:value={notes}
					placeholder="Modifications, dial-in notes, etc."
					aria-label="Notes"
					aria-describedby="grinder-notes-help"
				></textarea>
			</div>

			<div>
				<label class="form-label" for="grinder-link">Link</label>
				<input
					id="grinder-link"
					name="link"
					type="url"
					class="w-full form-input-lg"
					bind:value={link}
					placeholder="https://example.com/grinder"
					aria-label="Link"
					autocomplete="url"
				/>
			</div>

			<div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-3 pt-2">
				<a href={isEdit && grinder ? `/grinders/${encodeURIComponent(actor())}/${encodeURIComponent(grinder.rkey)}` : "/my-coffee"} class="btn-secondary text-center">Cancel</a>
				<button type="submit" class="btn-primary" disabled={submitting}>
					{submitting ? "Saving..." : isEdit ? "Save Changes" : "Add Grinder"}
				</button>
			</div>
		</form>
	</div>
</div>
