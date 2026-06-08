<script lang="ts">
  import { onMount } from "svelte";

  type RoasterRow = {
    rkey: string;
    name: string;
  };

  let { target }: { target: HTMLElement } = $props();

  let allRoasters = $state<RoasterRow[]>([]);
  let filtered = $state<RoasterRow[]>([]);
  let query = $state("");
  let selectedRKey = $state("");
  let newRoasterName = $state("");
  let newRoasterLocation = $state("");
  let newRoasterWebsite = $state("");
  let showDropdown = $state(false);
  let showDetails = $state(false);

  const exactMatch = $derived.by(() => {
    const q = query.trim().toLowerCase();
    return allRoasters.some((roaster) => roaster.name.toLowerCase() === q);
  });

  function filter() {
    const q = query.trim().toLowerCase();
    selectedRKey = "";
    newRoasterName = "";
    if (!q) {
      filtered = allRoasters.slice(0, 10);
      return;
    }
    filtered = allRoasters.filter((roaster) =>
      roaster.name.toLowerCase().includes(q),
    );
  }

  function selectRoaster(roaster: RoasterRow) {
    selectedRKey = roaster.rkey;
    newRoasterName = "";
    newRoasterLocation = "";
    newRoasterWebsite = "";
    query = roaster.name;
    showDropdown = false;
    showDetails = false;
  }

  function startCreate() {
    newRoasterName = query.trim();
    selectedRKey = "";
    showDropdown = false;
    showDetails = true;
  }

  function cancelCreate() {
    newRoasterName = "";
    newRoasterLocation = "";
    newRoasterWebsite = "";
    showDetails = false;
  }

  function clear() {
    query = "";
    selectedRKey = "";
    newRoasterName = "";
    newRoasterLocation = "";
    newRoasterWebsite = "";
    showDetails = false;
    filtered = allRoasters.slice(0, 10);
  }

  onMount(() => {
    try {
      allRoasters = JSON.parse(target.dataset.roasters || "[]");
    } catch {
      allRoasters = [];
    }
    selectedRKey = target.dataset.initialRkey || "";
    query = target.dataset.initialName || "";
    filtered = allRoasters.slice(0, 10);
  });
</script>

{#if !showDetails}
  <input
    type="text"
    bind:value={query}
    oninput={filter}
    onfocus={() => (showDropdown = true)}
    onblur={() => window.setTimeout(() => (showDropdown = false), 150)}
    onkeydown={(event) => {
      if (event.key === "Escape") {
        event.preventDefault();
        showDropdown = false;
      }
    }}
    placeholder="Search or create roaster"
    class="w-full form-input"
    autocomplete="off"
  />
{/if}

<input type="hidden" name="roaster_rkey" value={selectedRKey} />
<input type="hidden" name="new_roaster_name" value={newRoasterName} />
<input type="hidden" name="new_roaster_location" value={newRoasterLocation} />
<input type="hidden" name="new_roaster_website" value={newRoasterWebsite} />

{#if (selectedRKey || newRoasterName) && !showDetails}
  <button
    type="button"
    onclick={clear}
    class="absolute right-2 top-1/2 -translate-y-1/2 text-placeholder hover:text-muted text-sm"
    aria-label="Clear roaster"
  >
    &times;
  </button>
{/if}

{#if showDropdown && !showDetails}
  <div
    class="absolute z-10 w-full mt-1 max-h-48 overflow-y-auto rounded-lg shadow-lg"
    style="background: var(--card-bg, #fff); border: 1px solid var(--surface-border, #d4c4a8);"
  >
    {#each filtered as roaster}
      <button
        type="button"
        class="block w-full text-left px-3 py-2 cursor-pointer text-sm hover:bg-brown-100"
        onmousedown={(event) => {
          event.preventDefault();
          selectRoaster(roaster);
        }}
      >
        {roaster.name}
      </button>
    {/each}
    {#if query.trim() && !exactMatch}
      <button
        type="button"
        class="block w-full text-left px-3 py-2 cursor-pointer text-sm font-medium border-t hover:bg-brown-100"
        style="border-color: var(--surface-border, #d4c4a8);"
        onmousedown={(event) => {
          event.preventDefault();
          startCreate();
        }}
      >
        Create "{query.trim()}"
      </button>
    {/if}
  </div>
{/if}

{#if showDetails}
  <div
    class="p-3 rounded-lg space-y-2"
    style="background: var(--surface-bg); border: 1px solid var(--surface-border);"
  >
    <p class="text-sm font-medium text-primary">
      New roaster: <span class="font-semibold">{newRoasterName}</span>
    </p>
    <input
      type="text"
      bind:value={newRoasterLocation}
      placeholder="Location (optional)"
      class="w-full form-input text-sm"
    />
    <input
      type="url"
      bind:value={newRoasterWebsite}
      placeholder="Website (optional)"
      class="w-full form-input text-sm"
    />
    <button
      type="button"
      onclick={cancelCreate}
      class="text-xs text-faint hover:text-emphasis"
    >
      Cancel
    </button>
  </div>
{/if}
