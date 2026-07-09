<script lang="ts">
	import { goto } from "$app/navigation";
	import BackButton from "./BackButton.svelte";
	import { APIError } from "$lib/api/client";
	import { createRoaster, updateRoaster } from "$lib/api/entities";
	import { appCache } from "$lib/stores/appCache";
	import { session } from "$lib/stores/session";
	import { pushToast } from "$lib/stores/toasts";
	import type { Roaster } from "$lib/types/entity_view";

	type Props = {
		roaster: Roaster | null;
		isEdit: boolean;
	};

	let { roaster, isEdit }: Props = $props();

	// svelte-ignore state_referenced_locally
	let name = $state(roaster?.name ?? "");
	// svelte-ignore state_referenced_locally
	let location = $state(roaster?.location ?? "");
	// svelte-ignore state_referenced_locally
	let website = $state(roaster?.website ?? "");
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
			location: location.trim(),
			website: website.trim(),
			...(roaster?.source_ref ? { source_ref: roaster.source_ref } : {}),
		};

		try {
			const saved = isEdit && roaster
				? await updateRoaster(fetch, roaster.rkey, input)
				: await createRoaster(fetch, input);
			appCache.invalidateCache();
			pushToast(isEdit ? "Roaster updated" : "Roaster added");
			const owner = actor();
			if (owner && saved.rkey) {
				await goto(`/roasters/${encodeURIComponent(owner)}/${encodeURIComponent(saved.rkey)}`);
				return;
			}
			await goto("/my-coffee");
		} catch (error) {
			if (error instanceof APIError) {
				nameError = error.fields?.name ?? "";
				formError = error.message;
			} else {
				formError = "Failed to save roaster";
			}
			pushToast(isEdit ? "Failed to update roaster" : "Failed to add roaster");
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
				<p class="text-xs font-semibold uppercase tracking-wider text-faint">Roaster</p>
				<h1 class="text-2xl font-semibold text-primary">{isEdit ? "Edit Roaster" : "Add a Roaster"}</h1>
			</div>
		</div>

		<form class="space-y-6" novalidate onsubmit={submit}>
			{#if formError}
				<div class="alert-error" role="alert">{formError}</div>
			{/if}

			<div>
				<label class="form-label" for="roaster-name">Name <span class="text-red-500" aria-hidden="true">*</span></label>
				<p id="roaster-name-help" class="text-sm text-muted mb-2">The name shown on bags and in your coffee journal.</p>
				<input
					id="roaster-name"
					name="name"
					type="text"
					class="w-full form-input-lg"
					bind:value={name}
					aria-label="Name"
					required
					autocomplete="organization"
					aria-invalid={nameError !== ""}
					aria-describedby={nameError ? "roaster-name-help roaster-name-error" : "roaster-name-help"}
					oninput={() => { if (nameError && name.trim()) nameError = ""; }}
				/>
				{#if nameError}
					<p id="roaster-name-error" class="text-sm text-red-600 mt-2" role="alert">{nameError}</p>
				{/if}
			</div>

			<div>
				<label class="form-label" for="roaster-location">Location</label>
				<p id="roaster-location-help" class="text-sm text-muted mb-2">City, region, or country.</p>
				<input
					id="roaster-location"
					name="location"
					type="text"
					class="w-full form-input-lg"
					bind:value={location}
					aria-label="Location"
					autocomplete="address-level2"
					aria-describedby="roaster-location-help"
				/>
			</div>

			<div>
				<label class="form-label" for="roaster-website">Website</label>
				<p id="roaster-website-help" class="text-sm text-muted mb-2">A link to the roaster's website.</p>
				<input
					id="roaster-website"
					name="website"
					type="url"
					class="w-full form-input-lg"
					bind:value={website}
					aria-label="Website"
					placeholder="https://example.com"
					autocomplete="url"
					aria-describedby="roaster-website-help"
				/>
			</div>

			<div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-3 pt-2">
				<a href={isEdit && roaster ? `/roasters/${encodeURIComponent(actor())}/${encodeURIComponent(roaster.rkey)}` : "/my-coffee"} class="btn-secondary text-center">Cancel</a>
				<button type="submit" class="btn-primary" disabled={submitting}>
					{submitting ? "Saving..." : isEdit ? "Save Changes" : "Add Roaster"}
				</button>
			</div>
		</form>
	</div>
</div>
