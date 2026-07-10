<script lang="ts">
  import EntityCombo from "./EntityCombo.svelte";
  import { APIError } from "$lib/api/client";
  import {
    createBean,
    createBrewer,
    createGrinder,
    createRoaster,
  } from "$lib/api/entities";
  import { appCache } from "$lib/stores/appCache";
  import { pushToast } from "$lib/stores/toasts";
  import type { Roaster } from "$lib/types/entity_view";

  type Props = {
    kind: string;
    roasters: Roaster[];
    onclose: () => void;
    onsaved: () => Promise<void>;
  };
  let { kind, roasters: _roasters, onclose, onsaved }: Props = $props();
  let name = $state("");
  let origin = $state("");
  let roasterRKey = $state("");
  let brewerType = $state("");
  let grinderType = $state("");
  let burrType = $state("");
  let location = $state("");
  let website = $state("");
  let link = $state("");
  let description = $state("");
  let notes = $state("");
  let error = $state("");
  let saving = $state(false);
  let title = $derived(kind[0].toUpperCase() + kind.slice(1));
  async function submit(event: SubmitEvent) {
    event.preventDefault();
    if (saving) return;
    error = "";
    if (!name.trim() || (kind === "bean" && !origin.trim())) {
      error =
        kind === "bean" && !origin.trim()
          ? "Name and origin are required."
          : "Name is required.";
      return;
    }
    saving = true;
    try {
      switch (kind) {
        case "brewer":
          await createBrewer(fetch, {
            name: name.trim(),
            brewer_type: brewerType,
            description: description.trim(),
            link: link.trim(),
          });
          break;
        case "roaster":
          await createRoaster(fetch, {
            name: name.trim(),
            location: location.trim(),
            website: website.trim(),
          });
          break;
        case "bean":
          await createBean(fetch, {
            name: name.trim(),
            origin: origin.trim(),
            variety: "",
            roast_level: "",
            process: "",
            description: "",
            notes: "",
            link: "",
            roaster_rkey: roasterRKey,
            closed: false,
          });
          break;
        case "grinder":
          await createGrinder(fetch, {
            name: name.trim(),
            grinder_type: grinderType,
            burr_type: burrType,
            notes: notes.trim(),
            link: link.trim(),
          });
          break;
      }
      appCache.invalidateCache();
      pushToast(`${title} added`);
      await onsaved();
    } catch (cause) {
      error =
        cause instanceof APIError
          ? cause.message
          : `Failed to add ${title.toLowerCase()}.`;
    } finally {
      saving = false;
    }
  }
</script>

<div class="station-drawer" data-drawer data-kind={kind}>
  <header class="station-drawer-head">
    <div class="station-drawer-eyebrow">adding</div>
    <h3 class="station-drawer-title">{title}</h3>
    <button
      type="button"
      class="station-drawer-close"
      onclick={onclose}
      aria-label="Cancel"
      ><svg
        width="18"
        height="18"
        viewBox="0 0 20 20"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        aria-hidden="true"><path d="M5 5l10 10M15 5l-10 10"></path></svg
      ></button
    >
  </header>
  <form class="station-drawer-form" novalidate onsubmit={submit}>
    {#if error}<div class="station-drawer-error" role="alert">{error}</div>{/if}
    <div class="form-fieldset">
      <div class="form-fieldset-label">Essentials</div>
      <input
        class="w-full form-input"
        bind:value={name}
        placeholder="Name *"
        required
        autocomplete="off"
      />
      {#if kind === "bean"}<span class="form-label" id="station-roaster-label"
          >Roaster</span
        ><EntityCombo
          entityType="roaster"
          apiEndpoint="/api/roasters"
          suggestEndpoint="/api/suggestions/roasters"
          inputName="roaster_rkey"
          placeholder="Search or create roaster"
          sectionLabel="Your roasters"
          bind:rkey={roasterRKey}
          ariaLabel="Roaster"
        /><input
          class="w-full form-input"
          bind:value={origin}
          placeholder="Origin *"
          required
        />
      {:else if kind === "brewer"}<select
          class="w-full form-select"
          bind:value={brewerType}
          ><option value="">Select type...</option><option value="pourover"
            >Pour-over</option
          ><option value="espresso">Espresso</option><option value="immersion"
            >Immersion</option
          ><option value="mokapot">Moka Pot</option><option value="coldbrew"
            >Cold Brew</option
          ><option value="cupping">Cupping</option><option value="other"
            >Other</option
          ></select
        ><input
          class="w-full form-input"
          type="url"
          bind:value={link}
          placeholder="Link"
        />
      {:else if kind === "grinder"}<select
          class="w-full form-input"
          bind:value={grinderType}
          required
          ><option value="">Select Grinder Type *</option><option value="Hand"
            >Hand</option
          ><option value="Electric">Electric</option><option
            value="Portable Electric">Portable Electric</option
          ></select
        >
      {:else if kind === "roaster"}<input
          class="w-full form-input"
          bind:value={location}
          placeholder="Location"
        /><input
          class="w-full form-input"
          type="url"
          bind:value={website}
          placeholder="Website"
        />{/if}
    </div>
    {#if kind === "brewer" || kind === "grinder"}<div
        class="form-divider"
      ></div>
      <div class="form-fieldset">
        <div class="form-fieldset-label">
          Details <span class="form-optional-hint">(optional)</span>
        </div>
        {#if kind === "brewer"}<textarea
            class="w-full form-textarea"
            bind:value={description}
            placeholder="Description"
            rows="3"
          ></textarea>{/if}{#if kind === "grinder"}<select
            class="w-full form-input"
            bind:value={burrType}
            ><option value="">Select Burr Type</option><option value="Conical"
              >Conical</option
            ><option value="Flat">Flat</option></select
          ><input
            class="w-full form-input"
            type="url"
            bind:value={link}
            placeholder="Link"
          /><textarea
            class="w-full form-textarea"
            bind:value={notes}
            placeholder="Notes"
            rows="3"
          ></textarea>{/if}
      </div>{/if}
    <div class="station-drawer-actions">
      <button
        type="submit"
        class="btn-primary station-drawer-save"
        disabled={saving}>{saving ? "Saving..." : `Save ${title}`}</button
      ><button
        type="button"
        class="btn-secondary"
        onclick={onclose}
        disabled={saving}>Cancel</button
      >
    </div>
  </form>
</div>
