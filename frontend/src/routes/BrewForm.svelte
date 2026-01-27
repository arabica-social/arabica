<script>
  import { onMount } from "svelte";
  import { authStore } from "../stores/auth.js";
  import { cacheStore } from "../stores/cache.js";
  import { navigate, back } from "../lib/router.js";
  import { api } from "../lib/api.js";
  import Modal from "../components/Modal.svelte";

  export let id = null; // RKey for edit mode
  export let mode = "create"; // 'create' or 'edit'

  let form = {
    bean_rkey: "",
    coffee_amount: "",
    grinder_rkey: "",
    grind_size: "",
    brewer_rkey: "",
    water_amount: "",
    water_temp: "",
    brew_time: "",
    notes: "",
    rating: 5,
  };

  let pours = [];
  let loading = true;
  let saving = false;
  let error = null;

  // Modal states
  let showBeanModal = false;
  let showRoasterModal = false;
  let showGrinderModal = false;
  let showBrewerModal = false;

  // Modal forms
  let beanForm = {
    name: "",
    origin: "",
    roast_level: "",
    process: "",
    description: "",
    roaster_rkey: "",
  };
  let roasterForm = { name: "", location: "", website: "", description: "" };
  let grinderForm = { name: "", grinder_type: "", burr_type: "", notes: "" };
  let brewerForm = { name: "", brewer_type: "", description: "" };

  $: beans = $cacheStore.beans || [];
  $: roasters = $cacheStore.roasters || [];
  $: grinders = $cacheStore.grinders || [];
  $: brewers = $cacheStore.brewers || [];
  $: isAuthenticated = $authStore.isAuthenticated;

  onMount(async () => {
    if (!isAuthenticated) {
      navigate("/login");
      return;
    }

    await cacheStore.load();

    if (mode === "edit" && id) {
      // Load brew for editing
      const brews = $cacheStore.brews || [];
      const brew = brews.find((b) => b.rkey === id);

      if (brew) {
        form = {
          bean_rkey: brew.bean_rkey || "",
          coffee_amount: brew.coffee_amount || "",
          grinder_rkey: brew.grinder_rkey || "",
          grind_size: brew.grind_size || "",
          brewer_rkey: brew.brewer_rkey || "",
          water_amount: brew.water_amount || "",
          water_temp: brew.temperature || "",
          brew_time: brew.time_seconds || "",
          notes: brew.tasting_notes || "",
          rating: brew.rating || 5,
        };

        pours = brew.pours ? JSON.parse(JSON.stringify(brew.pours)) : [];
      } else {
        error = "Brew not found";
      }
    }

    loading = false;
  });

  function addPour() {
    pours = [...pours, { water_amount: 0, time_seconds: 0 }];
  }

  function removePour(index) {
    pours = pours.filter((_, i) => i !== index);
  }

  async function handleSubmit() {
    // Validate required fields
    if (!form.bean_rkey || form.bean_rkey === "") {
      error = "Please select a coffee bean";
      return;
    }

    saving = true;
    error = null;

    try {
      const payload = {
        bean_rkey: form.bean_rkey,
        method: form.method || "",
        temperature: form.water_temp ? parseFloat(form.water_temp) : 0,
        water_amount: form.water_amount ? parseFloat(form.water_amount) : 0,
        coffee_amount: form.coffee_amount ? parseFloat(form.coffee_amount) : 0,
        time_seconds: form.brew_time ? parseFloat(form.brew_time) : 0,
        grind_size: form.grind_size || "",
        grinder_rkey: form.grinder_rkey || "",
        brewer_rkey: form.brewer_rkey || "",
        tasting_notes: form.notes || "",
        rating: form.rating ? parseInt(form.rating) : 0,
        pours: pours.filter((p) => p.water_amount && p.time_seconds), // Only include completed pours
      };

      if (mode === "edit") {
        await api.put(`/brews/${id}`, payload);
      } else {
        await api.post("/brews", payload);
      }

      await cacheStore.invalidate();
      navigate("/brews");
    } catch (err) {
      error = err.message;
      saving = false;
    }
  }

  // Entity creation handlers
  async function saveBeanModal() {
    try {
      const result = await api.post("/api/beans", beanForm);
      await cacheStore.invalidate();
      form.bean_rkey = result.rkey;
      showBeanModal = false;
      beanForm = {
        name: "",
        origin: "",
        roast_level: "",
        process: "",
        description: "",
        roaster_rkey: "",
      };
    } catch (err) {
      alert("Failed to create bean: " + err.message);
    }
  }

  async function saveRoasterModal() {
    try {
      const result = await api.post("/api/roasters", roasterForm);
      await cacheStore.invalidate();
      beanForm.roaster_rkey = result.rkey;
      showRoasterModal = false;
      roasterForm = { name: "", location: "", website: "", description: "" };
    } catch (err) {
      alert("Failed to create roaster: " + err.message);
    }
  }

  async function saveGrinderModal() {
    try {
      const result = await api.post("/api/grinders", grinderForm);
      await cacheStore.invalidate();
      form.grinder_rkey = result.rkey;
      showGrinderModal = false;
      grinderForm = { name: "", grinder_type: "", burr_type: "", notes: "" };
    } catch (err) {
      alert("Failed to create grinder: " + err.message);
    }
  }

  async function saveBrewerModal() {
    try {
      const result = await api.post("/api/brewers", brewerForm);
      await cacheStore.invalidate();
      form.brewer_rkey = result.rkey;
      showBrewerModal = false;
      brewerForm = { name: "", brewer_type: "", description: "" };
    } catch (err) {
      alert("Failed to create brewer: " + err.message);
    }
  }
</script>

<svelte:head>
  <title>{mode === "edit" ? "Edit Brew" : "New Brew"} - Arabica</title>
</svelte:head>

<div class="max-w-2xl mx-auto">
  {#if loading}
    <div class="text-center py-12">
      <div
        class="animate-spin rounded-full h-12 w-12 border-b-2 border-brown-800 mx-auto"
      ></div>
      <p class="mt-4 text-brown-700">Loading...</p>
    </div>
  {:else}
    <div
      class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-4 md:p-8 border border-brown-300"
    >
      <!-- Header with Back Button -->
      <div class="flex items-center gap-3 mb-6">
        <button
          on:click={() => back()}
          class="inline-flex items-center text-brown-700 hover:text-brown-900 font-medium transition-colors cursor-pointer"
        >
          <svg
            class="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
            xmlns="http://www.w3.org/2000/svg"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M10 19l-7-7m0 0l7-7m-7 7h18"
            ></path>
          </svg>
        </button>
        <h2 class="text-3xl font-bold text-brown-900">
          {mode === "edit" ? "Edit Brew" : "New Brew"}
        </h2>
      </div>

      {#if error}
        <div
          class="mb-4 bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded"
        >
          {error}
        </div>
      {/if}

      <form
        on:submit|preventDefault={handleSubmit}
        class="space-y-4 md:space-y-6"
      >
        <!-- Bean Selection -->
        <div>
          <label
            for="bean-select"
            class="block text-sm font-medium text-brown-900 mb-2"
            >Coffee Bean *</label
          >
          <div class="flex gap-2">
            <select
              id="bean-select"
              bind:value={form.bean_rkey}
              required
              class="flex-1 rounded-lg border-2 border-brown-300 shadow-sm focus:border-brown-600 focus:ring-brown-600 text-base py-2 md:py-3 px-3 md:px-4 truncate max-w-full bg-white"
            >
              <option value="">Select a bean...</option>
              {#each beans as bean}
                <option value={bean.rkey}>
                  {bean.name || bean.origin} ({bean.origin} - {bean.roast_level})
                </option>
              {/each}
            </select>
            <button
              type="button"
              on:click={() => (showBeanModal = true)}
              class="bg-brown-300 text-brown-900 px-4 py-2 rounded-lg hover:bg-brown-400 font-medium transition-colors"
            >
              + New
            </button>
          </div>
        </div>

        <!-- Coffee Amount -->
        <div>
          <label
            for="coffee-amount"
            class="block text-sm font-medium text-brown-900 mb-2"
            >Coffee Amount (grams)</label
          >
          <input
            id="coffee-amount"
            type="number"
            bind:value={form.coffee_amount}
            step="0.1"
            placeholder="e.g. 18"
            class="w-full rounded-lg border-2 border-brown-300 shadow-sm focus:border-brown-600 focus:ring-brown-600 text-base py-2 md:py-3 px-3 md:px-4 bg-white"
          />
          <p class="text-sm text-brown-700 mt-1">
            Amount of ground coffee used
          </p>
        </div>

        <!-- Grinder -->
        <div>
          <label
            for="grinder-select"
            class="block text-sm font-medium text-brown-900 mb-2">Grinder</label
          >
          <div class="flex gap-2">
            <select
              id="grinder-select"
              bind:value={form.grinder_rkey}
              class="flex-1 rounded-lg border-2 border-brown-300 shadow-sm focus:border-brown-600 focus:ring-brown-600 text-base py-2 md:py-3 px-3 md:px-4 truncate max-w-full bg-white"
            >
              <option value="">Select a grinder...</option>
              {#each grinders as grinder}
                <option value={grinder.rkey}>{grinder.name}</option>
              {/each}
            </select>
            <button
              type="button"
              on:click={() => (showGrinderModal = true)}
              class="bg-brown-300 text-brown-900 px-4 py-2 rounded-lg hover:bg-brown-400 font-medium transition-colors"
            >
              + New
            </button>
          </div>
        </div>

        <!-- Grind Size -->
        <div>
          <label
            for="grind-size"
            class="block text-sm font-medium text-brown-900 mb-2"
            >Grind Size</label
          >
          <input
            id="grind-size"
            type="text"
            bind:value={form.grind_size}
            placeholder="e.g. 18, Medium, 3.5, Fine"
            class="w-full rounded-lg border-2 border-brown-300 shadow-sm focus:border-brown-600 focus:ring-brown-600 text-base py-2 md:py-3 px-3 md:px-4 bg-white"
          />
          <p class="text-sm text-brown-700 mt-1">
            Enter a number (grinder setting) or description (e.g. "Medium",
            "Fine")
          </p>
        </div>

        <!-- Brew Method -->
        <div>
          <label
            for="brewer-select"
            class="block text-sm font-medium text-brown-900 mb-2"
            >Brew Method</label
          >
          <div class="flex gap-2">
            <select
              id="brewer-select"
              bind:value={form.brewer_rkey}
              class="flex-1 rounded-lg border-2 border-brown-300 shadow-sm focus:border-brown-600 focus:ring-brown-600 text-base py-2 md:py-3 px-3 md:px-4 truncate max-w-full bg-white"
            >
              <option value="">Select brew method...</option>
              {#each brewers as brewer}
                <option value={brewer.rkey}>{brewer.name}</option>
              {/each}
            </select>
            <button
              type="button"
              on:click={() => (showBrewerModal = true)}
              class="bg-brown-300 text-brown-900 px-4 py-2 rounded-lg hover:bg-brown-400 font-medium transition-colors"
            >
              + New
            </button>
          </div>
        </div>

        <!-- Water Amount -->
        <div>
          <label
            for="water-amount"
            class="block text-sm font-medium text-brown-900 mb-2"
            >Water Amount (optional)</label
          >
          <input
            id="water-amount"
            type="number"
            bind:value={form.water_amount}
            step="1"
            placeholder="e.g. 300"
            class="w-full rounded-lg border-2 border-brown-300 shadow-sm focus:border-brown-600 focus:ring-brown-600 text-base py-2 md:py-3 px-3 md:px-4 bg-white"
          />
        </div>

        <!-- Pours -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <span class="block text-sm font-medium text-brown-900"
              >Pour Schedule (Optional)</span
            >
            <button
              type="button"
              on:click={addPour}
              class="text-sm bg-brown-300 text-brown-900 px-3 py-1 rounded hover:bg-brown-400 font-medium transition-colors"
            >
              + Add Pour
            </button>
          </div>

          {#if pours.length > 0}
            <div class="space-y-2">
              {#each pours as pour, i}
                <div
                  class="flex gap-2 items-center bg-brown-50 p-2 md:p-3 rounded-lg border border-brown-200"
                >
                  <span
                    class="text-xs md:text-sm font-medium text-brown-700 min-w-[50px] md:min-w-[60px]"
                    >Pour {i + 1}:</span
                  >
                  <input
                    type="number"
                    bind:value={pour.water_amount}
                    placeholder="g"
                    class="w-16 md:w-20 rounded border border-brown-300 px-2 py-2 text-sm"
                  />
                  <input
                    type="number"
                    bind:value={pour.time_seconds}
                    placeholder="sec"
                    class="w-16 md:w-20 rounded border border-brown-300 px-2 py-2 text-sm"
                  />
                  <button
                    type="button"
                    on:click={() => removePour(i)}
                    class="text-red-600 hover:text-red-800 font-medium px-2 flex-shrink-0"
                  >
                    ✕
                  </button>
                </div>
              {/each}
            </div>
          {/if}
        </div>

        <!-- Water Temperature -->
        <div>
          <label
            for="water-temp"
            class="block text-sm font-medium text-brown-900 mb-2"
            >Water Temperature (°C)</label
          >
          <input
            id="water-temp"
            type="number"
            bind:value={form.water_temp}
            step="0.1"
            placeholder="e.g. 93"
            class="w-full rounded-lg border-2 border-brown-300 shadow-sm focus:border-brown-600 focus:ring-brown-600 text-base py-2 md:py-3 px-3 md:px-4 bg-white"
          />
        </div>

        <!-- Brew Time -->
        <div>
          <label
            for="brew-time"
            class="block text-sm font-medium text-brown-900 mb-2"
            >Total Brew Time (seconds)</label
          >
          <input
            id="brew-time"
            type="number"
            bind:value={form.brew_time}
            step="1"
            placeholder="e.g. 210"
            class="w-full rounded-lg border-2 border-brown-300 shadow-sm focus:border-brown-600 focus:ring-brown-600 text-base py-2 md:py-3 px-3 md:px-4 bg-white"
          />
        </div>

        <!-- Rating -->
        <div>
          <label
            for="rating"
            class="block text-sm font-medium text-brown-900 mb-2"
          >
            Rating: <span class="font-bold">{form.rating}/10</span>
          </label>
          <input
            id="rating"
            type="range"
            bind:value={form.rating}
            min="0"
            max="10"
            step="1"
            class="w-full h-2 bg-brown-200 rounded-lg appearance-none cursor-pointer accent-brown-700"
          />
          <div class="flex justify-between text-xs text-brown-600 mt-1">
            <span>0</span>
            <span>10</span>
          </div>
        </div>

        <!-- Notes -->
        <div>
          <label
            for="notes"
            class="block text-sm font-medium text-brown-900 mb-2"
            >Tasting Notes</label
          >
          <textarea
            id="notes"
            bind:value={form.notes}
            rows="4"
            placeholder="Describe the flavor, aroma, body, etc."
            class="w-full rounded-lg border-2 border-brown-300 shadow-sm focus:border-brown-600 focus:ring-brown-600 text-base py-2 md:py-3 px-3 md:px-4 bg-white"
          ></textarea>
        </div>

        <!-- Submit Button -->
        <div class="flex gap-3">
          <button
            type="submit"
            disabled={saving}
            class="flex-1 bg-gradient-to-r from-brown-700 to-brown-800 text-white py-3 px-6 rounded-lg hover:from-brown-800 hover:to-brown-900 transition-all font-semibold shadow-lg disabled:opacity-50"
          >
            {saving
              ? "Saving..."
              : mode === "edit"
                ? "Update Brew"
                : "Save Brew"}
          </button>
          <button
            type="button"
            on:click={() => back()}
            class="px-6 py-3 border-2 border-brown-300 text-brown-700 rounded-lg hover:bg-brown-100 font-semibold transition-colors"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  {/if}
</div>

<!-- Modals -->
<Modal
  bind:isOpen={showBeanModal}
  title="Add New Bean"
  onSave={saveBeanModal}
  onCancel={() => (showBeanModal = false)}
>
  <div class="space-y-4">
    <div>
      <label
        for="bean-name"
        class="block text-sm font-medium text-gray-700 mb-1">Name</label
      >
      <input
        id="bean-name"
        type="text"
        bind:value={beanForm.name}
        class="w-full rounded border-gray-300 px-3 py-2"
      />
    </div>
    <div>
      <label
        for="bean-origin"
        class="block text-sm font-medium text-gray-700 mb-1">Origin *</label
      >
      <input
        id="bean-origin"
        type="text"
        bind:value={beanForm.origin}
        required
        class="w-full rounded border-gray-300 px-3 py-2"
      />
    </div>
    <div>
      <label
        for="bean-roast-level"
        class="block text-sm font-medium text-gray-700 mb-1"
        >Roast Level *</label
      >
      <select
        id="bean-roast-level"
        bind:value={beanForm.roast_level}
        required
        class="w-full rounded border-gray-300 px-3 py-2"
      >
        <option value="">Select...</option>
        <option value="Light">Light</option>
        <option value="Medium-Light">Medium-Light</option>
        <option value="Medium">Medium</option>
        <option value="Medium-Dark">Medium-Dark</option>
        <option value="Dark">Dark</option>
      </select>
    </div>
    <div>
      <label
        for="bean-roaster"
        class="block text-sm font-medium text-gray-700 mb-1">Roaster</label
      >
      <div class="flex gap-2">
        <select
          id="bean-roaster"
          bind:value={beanForm.roaster_rkey}
          class="flex-1 rounded border-gray-300 px-3 py-2"
        >
          <option value="">Select...</option>
          {#each roasters as roaster}
            <option value={roaster.rkey}>{roaster.name}</option>
          {/each}
        </select>
        <button
          type="button"
          on:click={() => (showRoasterModal = true)}
          class="bg-gray-200 px-3 py-1 rounded hover:bg-gray-300 text-sm"
        >
          + New
        </button>
      </div>
    </div>
  </div>
</Modal>

<Modal
  bind:isOpen={showRoasterModal}
  title="Add New Roaster"
  onSave={saveRoasterModal}
  onCancel={() => (showRoasterModal = false)}
>
  <div class="space-y-4">
    <div>
      <label
        for="roaster-name"
        class="block text-sm font-medium text-gray-700 mb-1">Name *</label
      >
      <input
        id="roaster-name"
        type="text"
        bind:value={roasterForm.name}
        required
        class="w-full rounded border-gray-300 px-3 py-2"
      />
    </div>
    <div>
      <label
        for="roaster-location"
        class="block text-sm font-medium text-gray-700 mb-1">Location</label
      >
      <input
        id="roaster-location"
        type="text"
        bind:value={roasterForm.location}
        class="w-full rounded border-gray-300 px-3 py-2"
      />
    </div>
  </div>
</Modal>

<Modal
  bind:isOpen={showGrinderModal}
  title="Add New Grinder"
  onSave={saveGrinderModal}
  onCancel={() => (showGrinderModal = false)}
>
  <div class="space-y-4">
    <div>
      <label
        for="grinder-name"
        class="block text-sm font-medium text-gray-700 mb-1">Name *</label
      >
      <input
        id="grinder-name"
        type="text"
        bind:value={grinderForm.name}
        required
        class="w-full rounded border-gray-300 px-3 py-2"
      />
    </div>
    <div>
      <label
        for="grinder-type"
        class="block text-sm font-medium text-gray-700 mb-1">Type</label
      >
      <select
        id="grinder-type"
        bind:value={grinderForm.grinder_type}
        class="w-full rounded border-gray-300 px-3 py-2"
      >
        <option value="">Select...</option>
        <option value="Manual">Manual</option>
        <option value="Electric">Electric</option>
        <option value="Blade">Blade</option>
      </select>
    </div>
  </div>
</Modal>

<Modal
  bind:isOpen={showBrewerModal}
  title="Add New Brewer"
  onSave={saveBrewerModal}
  onCancel={() => (showBrewerModal = false)}
>
  <div class="space-y-4">
    <div>
      <label
        for="brewer-name"
        class="block text-sm font-medium text-gray-700 mb-1">Name *</label
      >
      <input
        id="brewer-name"
        type="text"
        bind:value={brewerForm.name}
        required
        class="w-full rounded border-gray-300 px-3 py-2"
      />
    </div>
    <div>
      <label
        for="brewer-type"
        class="block text-sm font-medium text-gray-700 mb-1">Type</label
      >
      <select
        id="brewer-type"
        bind:value={brewerForm.brewer_type}
        class="w-full rounded border-gray-300 px-3 py-2"
      >
        <option value="">Select...</option>
        <option value="Pour Over">Pour Over</option>
        <option value="French Press">French Press</option>
        <option value="Espresso">Espresso</option>
        <option value="Moka Pot">Moka Pot</option>
        <option value="Aeropress">Aeropress</option>
        <option value="Cold Brew">Cold Brew</option>
        <option value="Siphon">Siphon</option>
      </select>
    </div>
  </div>
</Modal>
