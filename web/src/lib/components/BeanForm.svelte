<script lang="ts">
	import { goto } from "$app/navigation";
	import BackButton from "./BackButton.svelte";
	import EntityCombo from "./EntityCombo.svelte";
	import { APIError } from "$lib/api/client";
	import { createBean, updateBean } from "$lib/api/entities";
	import { appCache } from "$lib/stores/appCache";
	import { session } from "$lib/stores/session";
	import { pushToast } from "$lib/stores/toasts";
	import type { Bean } from "$lib/types/entity_view";

	const ROAST_LEVELS = [
		"Ultra-Light",
		"Light",
		"Medium-Light",
		"Medium",
		"Medium-Dark",
		"Dark",
	];

	type Props = {
		bean: Bean | null;
		isEdit: boolean;
	};

	let { bean, isEdit }: Props = $props();

	// svelte-ignore state_referenced_locally
	let name = $state(bean?.name ?? "");
	// svelte-ignore state_referenced_locally
	let origin = $state(bean?.origin ?? "");
	// svelte-ignore state_referenced_locally
	let variety = $state(bean?.variety ?? "");
	// svelte-ignore state_referenced_locally
	let roastLevel = $state(bean?.roast_level ?? "");
	// svelte-ignore state_referenced_locally
	let roastDate = $state(bean?.roast_date ?? "");
	// svelte-ignore state_referenced_locally
	let process = $state(bean?.process ?? "");
	// svelte-ignore state_referenced_locally
	let description = $state(bean?.description ?? "");
	// svelte-ignore state_referenced_locally
	let notes = $state(bean?.notes ?? "");
	// svelte-ignore state_referenced_locally
	let link = $state(bean?.link ?? "");
	// svelte-ignore state_referenced_locally
	let roasterRKey = $state(bean?.roaster?.rkey ?? "");
	// svelte-ignore state_referenced_locally
	let rating = $state<number | undefined>(bean?.rating);
	// svelte-ignore state_referenced_locally
	let closed = $state(bean?.closed ?? false);

	let nameError = $state("");
	let originError = $state("");
	let formError = $state("");
	let submitting = $state(false);

	function validate(): boolean {
		nameError = name.trim() ? "" : "Name is required";
		originError = origin.trim() ? "" : "Origin is required";
		return nameError === "" && originError === "";
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
			origin: origin.trim(),
			variety: variety.trim(),
			roast_level: roastLevel,
			roast_date: roastDate || undefined,
			process: process.trim(),
			description: description.trim(),
			notes: notes.trim(),
			link: link.trim(),
			roaster_rkey: roasterRKey,
			rating,
			closed,
			...(bean?.source_ref ? { source_ref: bean.source_ref } : {}),
		};

		try {
			const saved = isEdit && bean
				? await updateBean(fetch, bean.rkey, input)
				: await createBean(fetch, input);
			appCache.invalidateCache();
			pushToast(isEdit ? "Bean updated" : "Bean added");
			const owner = actor();
			if (owner && saved.rkey) {
				await goto(`/beans/${encodeURIComponent(owner)}/${encodeURIComponent(saved.rkey)}`);
				return;
			}
			await goto("/my-coffee");
		} catch (error) {
			if (error instanceof APIError) {
				nameError = error.fields?.name ?? nameError;
				originError = error.fields?.origin ?? originError;
				formError = error.message;
			} else {
				formError = "Failed to save bean";
			}
			pushToast(isEdit ? "Failed to update bean" : "Failed to add bean");
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
				<p class="text-xs font-semibold uppercase tracking-wider text-faint">Bean</p>
				<h1 class="text-2xl font-semibold text-primary">{isEdit ? "Edit Bean" : "Add a Bean"}</h1>
			</div>
		</div>

		<form class="space-y-6" novalidate onsubmit={submit}>
			{#if formError}
				<div class="alert-error" role="alert">{formError}</div>
			{/if}

			<fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
				<legend class="text-sm font-semibold text-secondary px-2">Essentials</legend>

				<div>
					<label class="form-label" for="bean-name">Name <span class="text-red-500" aria-hidden="true">*</span></label>
					<p id="bean-name-help" class="text-sm text-muted mb-2">The coffee name shown in your journal.</p>
					<input
						id="bean-name"
						name="name"
						type="text"
						class="w-full form-input-lg"
						bind:value={name}
						placeholder="e.g. Ethiopia Gedeb"
						aria-label="Name"
						required
						autocomplete="off"
						aria-invalid={nameError !== ""}
						aria-describedby={nameError ? "bean-name-help bean-name-error" : "bean-name-help"}
						oninput={() => { if (nameError && name.trim()) nameError = ""; }}
					/>
					{#if nameError}
						<p id="bean-name-error" class="text-sm text-red-600 mt-2" role="alert">{nameError}</p>
					{/if}
				</div>

				<div>
					<span class="form-label" id="bean-roaster-label">Roaster</span>
					<p class="text-sm text-muted mb-2">Search your roasters or create a new one.</p>
					<EntityCombo
						entityType="roaster"
						apiEndpoint="/api/roasters"
						suggestEndpoint="/api/suggestions/roasters"
						inputName="roaster_rkey"
						placeholder="Search or create roaster"
						sectionLabel="Your roasters"
						bind:rkey={roasterRKey}
						ariaLabel="Roaster"
						allowCreate={true}
					/>
				</div>

				<div>
					<label class="form-label" for="bean-origin">Origin <span class="text-red-500" aria-hidden="true">*</span></label>
					<p id="bean-origin-help" class="text-sm text-muted mb-2">Country, region, or farm.</p>
					<input
						id="bean-origin"
						name="origin"
						type="text"
						class="w-full form-input-lg"
						bind:value={origin}
						placeholder="e.g. Colombia, Huila"
						aria-label="Origin"
						required
						autocomplete="off"
						aria-invalid={originError !== ""}
						aria-describedby={originError ? "bean-origin-help bean-origin-error" : "bean-origin-help"}
						oninput={() => { if (originError && origin.trim()) originError = ""; }}
					/>
					{#if originError}
						<p id="bean-origin-error" class="text-sm text-red-600 mt-2" role="alert">{originError}</p>
					{/if}
				</div>

				<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<div>
						<label class="form-label" for="bean-roast-level">Roast level</label>
						<select id="bean-roast-level" name="roast_level" class="w-full form-input-lg" bind:value={roastLevel}>
							<option value="">Select roast level</option>
							{#each ROAST_LEVELS as level}
								<option value={level}>{level}</option>
							{/each}
						</select>
					</div>
					<div>
						<label class="form-label" for="bean-roast-date">Roast date</label>
						<input
							id="bean-roast-date"
							name="roast_date"
							type="date"
							class="w-full form-input-lg"
							bind:value={roastDate}
							aria-label="Roast date"
						/>
					</div>
				</div>
			</fieldset>

			<fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
				<legend class="text-sm font-semibold text-secondary px-2">Origin details <span class="form-optional-hint">(optional)</span></legend>
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
					<div>
						<label class="form-label" for="bean-variety">Variety</label>
						<input
							id="bean-variety"
							name="variety"
							type="text"
							class="w-full form-input-lg"
							bind:value={variety}
							placeholder="e.g. SL28, Typica, Gesha"
							aria-label="Variety"
							autocomplete="off"
						/>
					</div>
					<div>
						<label class="form-label" for="bean-process">Process</label>
						<input
							id="bean-process"
							name="process"
							type="text"
							class="w-full form-input-lg"
							bind:value={process}
							placeholder="e.g. Washed, Natural, Honey"
							aria-label="Process"
							autocomplete="off"
						/>
					</div>
				</div>
			</fieldset>

			<fieldset class="space-y-6 border border-brown-200 rounded-lg p-4 min-w-0">
				<legend class="text-sm font-semibold text-secondary px-2">Details <span class="form-optional-hint">(optional)</span></legend>
				<div>
					<label class="form-label" for="bean-description">Description</label>
					<textarea
						id="bean-description"
						name="description"
						rows="3"
						class="w-full form-textarea"
						bind:value={description}
						placeholder="Roaster description, tasting notes, etc."
						aria-label="Description"
					></textarea>
				</div>
				<div>
					<label class="form-label" for="bean-link">Link</label>
					<input
						id="bean-link"
						name="link"
						type="url"
						class="w-full form-input-lg"
						bind:value={link}
						placeholder="https://roaster.example/beans/..."
						aria-label="Link"
						autocomplete="url"
					/>
				</div>
				<details class="form-details-disclosure" open={isEdit}>
					<summary class="text-sm font-semibold text-secondary cursor-pointer hover:text-primary transition-colors flex items-center gap-2">
						<span aria-hidden="true" class="form-details-triangle">▶</span>
						<span>Personal details <span class="form-optional-hint">(optional)</span></span>
					</summary>
					<div class="mt-3 space-y-4">
						<div>
							<label class="form-label" for="bean-notes">Personal notes</label>
							<textarea
								id="bean-notes"
								name="notes"
								rows="3"
								class="w-full form-textarea"
								bind:value={notes}
								placeholder="Your own notes about this bag"
								aria-label="Personal notes"
							></textarea>
						</div>
						<div>
							<label class="form-label" for="bean-rating">Rating</label>
							<div class="flex items-center gap-3">
								<input
									id="bean-rating"
									name="rating"
									type="range"
									min="1"
									max="10"
									bind:value={rating}
									class="w-full accent-brown-700"
									aria-label="Rating"
								/>
								<span class="text-sm font-medium text-emphasis min-w-[2.5rem]">{rating ?? "—"}</span>
							</div>
						</div>
						<div class="flex items-center gap-2">
							<input
								id="bean-closed-checkbox"
								name="closed"
								type="checkbox"
								bind:checked={closed}
								class="rounded-sm border-brown-300 text-emphasis focus:ring-brown-500"
							/>
							<label for="bean-closed-checkbox" class="text-sm text-primary">Bag is closed/finished</label>
						</div>
					</div>
				</details>
			</fieldset>

			<div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-3 pt-2">
				<a href={isEdit && bean ? `/beans/${encodeURIComponent(actor())}/${encodeURIComponent(bean.rkey)}` : "/my-coffee"} class="btn-secondary text-center">Cancel</a>
				<button type="submit" class="btn-primary" disabled={submitting}>
					{submitting ? "Saving..." : isEdit ? "Save Changes" : "Add Bean"}
				</button>
			</div>
		</form>
	</div>
</div>
