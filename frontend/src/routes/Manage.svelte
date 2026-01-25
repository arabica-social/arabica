<script>
  import { onMount } from "svelte";
  import { authStore } from "../stores/auth.js";
  import { cacheStore } from "../stores/cache.js";
  import { navigate } from "../lib/router.js";
  import { api } from "../lib/api.js";
  import Modal from "../components/Modal.svelte";

  let activeTab = "beans"; // beans, roasters, grinders, brewers
  let loading = true;

  // Modal states
  let showBeanModal = false;
  let showRoasterModal = false;
  let showGrinderModal = false;
  let showBrewerModal = false;

  // Edit states
  let editingBean = null;
  let editingRoaster = null;
  let editingGrinder = null;
  let editingBrewer = null;

  // Forms
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

    // Load active tab from localStorage
    const savedTab = localStorage.getItem("arabica_manage_tab");
    if (savedTab) {
      activeTab = savedTab;
    }

    await cacheStore.load();
    loading = false;
  });

  function setTab(tab) {
    activeTab = tab;
    localStorage.setItem("arabica_manage_tab", tab);
  }

  // Bean handlers
  function addBean() {
    editingBean = null;
    beanForm = {
      name: "",
      origin: "",
      roast_level: "",
      process: "",
      description: "",
      roaster_rkey: "",
    };
    showBeanModal = true;
  }

  function editBean(bean) {
    editingBean = bean;
    beanForm = {
      name: bean.name || "",
      origin: bean.origin || "",
      roast_level: bean.roast_level || "",
      process: bean.process || "",
      description: bean.description || "",
      roaster_rkey: bean.roaster_rkey || "",
    };
    showBeanModal = true;
  }

  async function saveBean() {
    try {
      console.log("Saving bean with data:", beanForm);
      if (editingBean) {
        console.log("Updating bean:", editingBean.rkey);
        await api.put(`/api/beans/${editingBean.rkey}`, beanForm);
      } else {
        console.log("Creating new bean");
        await api.post("/api/beans", beanForm);
      }
      await cacheStore.invalidate();
      showBeanModal = false;
    } catch (err) {
      console.error("Bean save error:", err);
      alert("Failed to save bean: " + err.message);
    }
  }

  async function deleteBean(rkey) {
    if (!confirm("Are you sure you want to delete this bean?")) return;
    try {
      await api.delete(`/api/beans/${rkey}`);
      await cacheStore.invalidate();
    } catch (err) {
      alert("Failed to delete bean: " + err.message);
    }
  }

  // Roaster handlers
  function addRoaster() {
    editingRoaster = null;
    roasterForm = { name: "", location: "", website: "", description: "" };
    showRoasterModal = true;
  }

  function editRoaster(roaster) {
    editingRoaster = roaster;
    roasterForm = {
      name: roaster.name || "",
      location: roaster.location || "",
      website: roaster.website || "",
      description: roaster.Description || "",
    };
    showRoasterModal = true;
  }

  async function saveRoaster() {
    try {
      if (editingRoaster) {
        await api.put(`/api/roasters/${editingRoaster.rkey}`, roasterForm);
      } else {
        await api.post("/api/roasters", roasterForm);
      }
      await cacheStore.invalidate();
      showRoasterModal = false;
    } catch (err) {
      alert("Failed to save roaster: " + err.message);
    }
  }

  async function deleteRoaster(rkey) {
    if (!confirm("Are you sure you want to delete this roaster?")) return;
    try {
      await api.delete(`/api/roasters/${rkey}`);
      await cacheStore.invalidate();
    } catch (err) {
      alert("Failed to delete roaster: " + err.message);
    }
  }

  // Grinder handlers
  function addGrinder() {
    editingGrinder = null;
    grinderForm = { name: "", grinder_type: "", burr_type: "", notes: "" };
    showGrinderModal = true;
  }

  function editGrinder(grinder) {
    editingGrinder = grinder;
    grinderForm = {
      name: grinder.name || "",
      grinder_type: grinder.grinder_type || "",
      burr_type: grinder.burr_type || "",
      notes: grinder.notes || "",
    };
    showGrinderModal = true;
  }

  async function saveGrinder() {
    try {
      if (editingGrinder) {
        await api.put(`/api/grinders/${editingGrinder.rkey}`, grinderForm);
      } else {
        await api.post("/api/grinders", grinderForm);
      }
      await cacheStore.invalidate();
      showGrinderModal = false;
    } catch (err) {
      alert("Failed to save grinder: " + err.message);
    }
  }

  async function deleteGrinder(rkey) {
    if (!confirm("Are you sure you want to delete this grinder?")) return;
    try {
      await api.delete(`/api/grinders/${rkey}`);
      await cacheStore.invalidate();
    } catch (err) {
      alert("Failed to delete grinder: " + err.message);
    }
  }

  // Brewer handlers
  function addBrewer() {
    editingBrewer = null;
    brewerForm = { name: "", brewer_type: "", description: "" };
    showBrewerModal = true;
  }

  function editBrewer(brewer) {
    editingBrewer = brewer;
    brewerForm = {
      name: brewer.name || "",
      brewer_type: brewer.brewer_type || "",
      description: brewer.description || "",
    };
    showBrewerModal = true;
  }

  async function saveBrewer() {
    try {
      if (editingBrewer) {
        await api.put(`/api/brewers/${editingBrewer.rkey}`, brewerForm);
      } else {
        await api.post("/api/brewers", brewerForm);
      }
      await cacheStore.invalidate();
      showBrewerModal = false;
    } catch (err) {
      alert("Failed to save brewer: " + err.message);
    }
  }

  async function deleteBrewer(rkey) {
    if (!confirm("Are you sure you want to delete this brewer?")) return;
    try {
      await api.delete(`/api/brewers/${rkey}`);
      await cacheStore.invalidate();
    } catch (err) {
      alert("Failed to delete brewer: " + err.message);
    }
  }
</script>

<svelte:head>
  <title>Manage - Arabica</title>
</svelte:head>

<div class="max-w-6xl mx-auto">
  <h1 class="text-3xl font-bold text-brown-900 mb-6">
    Manage Equipment & Beans
  </h1>

  {#if loading}
    <div class="text-center py-12">
      <div
        class="animate-spin rounded-full h-12 w-12 border-b-2 border-brown-800 mx-auto"
      ></div>
      <p class="mt-4 text-brown-700">Loading...</p>
    </div>
  {:else}
    <!-- Tab Navigation -->
    <div>
      <div
        class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-md mb-4 border border-brown-300"
      >
        <div class="flex border-b border-brown-300">
          <button
            on:click={() => setTab("beans")}
            class="flex-1 py-3 px-4 text-center font-medium transition-colors {activeTab ===
            'beans'
              ? 'border-b-2 border-brown-700 text-brown-900'
              : 'text-brown-600 hover:text-brown-800'}"
          >
            ☕ Beans
          </button>
          <button
            on:click={() => setTab("roasters")}
            class="flex-1 py-3 px-4 text-center font-medium transition-colors {activeTab ===
            'roasters'
              ? 'border-b-2 border-brown-700 text-brown-900'
              : 'text-brown-600 hover:text-brown-800'}"
          >
            🏭 Roasters
          </button>
          <button
            on:click={() => setTab("grinders")}
            class="flex-1 py-3 px-4 text-center font-medium transition-colors {activeTab ===
            'grinders'
              ? 'border-b-2 border-brown-700 text-brown-900'
              : 'text-brown-600 hover:text-brown-800'}"
          >
            ⚙️ Grinders
          </button>
          <button
            on:click={() => setTab("brewers")}
            class="flex-1 py-3 px-4 text-center font-medium transition-colors {activeTab ===
            'brewers'
              ? 'border-b-2 border-brown-700 text-brown-900'
              : 'text-brown-600 hover:text-brown-800'}"
          >
            🫖 Brewers
          </button>
        </div>
      </div>

      <!-- Tab Content -->
      <div>
        {#if activeTab === "beans"}
          <div class="flex justify-between items-center mb-4">
            <h2 class="text-xl font-bold text-brown-900">Coffee Beans</h2>
            <button
              on:click={addBean}
              class="bg-brown-700 text-white px-4 py-2 rounded-lg hover:bg-brown-800 font-medium"
            >
              + Add Bean
            </button>
          </div>

          {#if beans.length === 0}
            <div
              class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-8 text-center border border-brown-300"
            >
              <p class="text-brown-800 text-lg font-medium">
                No beans yet. Add your first bean!
              </p>
            </div>
          {:else}
            <div
              class="overflow-x-auto bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl border border-brown-300"
            >
              <table class="min-w-full divide-y divide-brown-300">
                <thead class="bg-brown-200/80">
                  <tr>
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >Name</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >📍 Origin</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >🔥 Roast</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >🏭 Roaster</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >📝 Description</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >Actions</th
                    >
                  </tr>
                </thead>
                <tbody class="bg-brown-50/60 divide-y divide-brown-200">
                  {#each beans as bean}
                    <tr class="hover:bg-brown-100/60 transition-colors">
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{bean.name || "-"}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{bean.origin}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{bean.roast_level}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{bean.roaster?.name || "-"}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-700 italic max-w-xs"
                        >{bean.description || "-"}</td
                      >
                      <td class="px-4 py-3 text-sm space-x-2">
                        <button
                          on:click={() => editBean(bean)}
                          class="text-brown-700 hover:text-brown-900 font-medium"
                        >
                          Edit
                        </button>
                        <button
                          on:click={() => deleteBean(bean.rkey)}
                          class="text-red-600 hover:text-red-800 font-medium"
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        {:else if activeTab === "roasters"}
          <div class="flex justify-between items-center mb-4">
            <h2 class="text-xl font-bold text-brown-900">Roasters</h2>
            <button
              on:click={addRoaster}
              class="bg-brown-700 text-white px-4 py-2 rounded-lg hover:bg-brown-800 font-medium"
            >
              + Add Roaster
            </button>
          </div>

          {#if roasters.length === 0}
            <div
              class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-8 text-center border border-brown-300"
            >
              <p class="text-brown-800 text-lg font-medium">
                No roasters yet. Add your first roaster!
              </p>
            </div>
          {:else}
            <div
              class="overflow-x-auto bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl border border-brown-300"
            >
              <table class="min-w-full divide-y divide-brown-300">
                <thead class="bg-brown-200/80">
                  <tr>
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >Name</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >📍 Location</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >🌐 Website</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >Actions</th
                    >
                  </tr>
                </thead>
                <tbody class="bg-brown-50/60 divide-y divide-brown-200">
                  {#each roasters as roaster}
                    <tr class="hover:bg-brown-100/60 transition-colors">
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{roaster.name}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{roaster.location || "-"}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-900">
                        {#if roaster.website}
                          <a
                            href={roaster.website}
                            target="_blank"
                            rel="noopener noreferrer"
                            class="text-brown-700 hover:underline font-medium"
                            >{roaster.website}</a
                          >
                        {:else}
                          -
                        {/if}
                      </td>
                      <td class="px-4 py-3 text-sm space-x-2">
                        <button
                          on:click={() => editRoaster(roaster)}
                          class="text-brown-700 hover:text-brown-900 font-medium"
                        >
                          Edit
                        </button>
                        <button
                          on:click={() => deleteRoaster(roaster.rkey)}
                          class="text-red-600 hover:text-red-800 font-medium"
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        {:else if activeTab === "grinders"}
          <div class="flex justify-between items-center mb-4">
            <h2 class="text-xl font-bold text-brown-900">Grinders</h2>
            <button
              on:click={addGrinder}
              class="bg-brown-700 text-white px-4 py-2 rounded-lg hover:bg-brown-800 font-medium"
            >
              + Add Grinder
            </button>
          </div>

          {#if grinders.length === 0}
            <div
              class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-8 text-center border border-brown-300"
            >
              <p class="text-brown-800 text-lg font-medium">
                No grinders yet. Add your first grinder!
              </p>
            </div>
          {:else}
            <div
              class="overflow-x-auto bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl border border-brown-300"
            >
              <table class="min-w-full divide-y divide-brown-300">
                <thead class="bg-brown-200/80">
                  <tr>
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >Name</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >🔧 Type</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >💎 Burr Type</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >📝 Notes</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >Actions</th
                    >
                  </tr>
                </thead>
                <tbody class="bg-brown-50/60 divide-y divide-brown-200">
                  {#each grinders as grinder}
                    <tr class="hover:bg-brown-100/60 transition-colors">
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{grinder.name}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{grinder.grinder_type || "-"}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{grinder.burr_type || "-"}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-700 italic max-w-xs"
                        >{grinder.notes || "-"}</td
                      >
                      <td class="px-4 py-3 text-sm space-x-2">
                        <button
                          on:click={() => editGrinder(grinder)}
                          class="text-brown-700 hover:text-brown-900 font-medium"
                        >
                          Edit
                        </button>
                        <button
                          on:click={() => deleteGrinder(grinder.rkey)}
                          class="text-red-600 hover:text-red-800 font-medium"
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        {:else if activeTab === "brewers"}
          <div class="flex justify-between items-center mb-4">
            <h2 class="text-xl font-bold text-brown-900">Brewers</h2>
            <button
              on:click={addBrewer}
              class="bg-brown-700 text-white px-4 py-2 rounded-lg hover:bg-brown-800 font-medium"
            >
              + Add Brewer
            </button>
          </div>

          {#if brewers.length === 0}
            <div
              class="bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl p-8 text-center border border-brown-300"
            >
              <p class="text-brown-800 text-lg font-medium">
                No brewers yet. Add your first brewer!
              </p>
            </div>
          {:else}
            <div
              class="overflow-x-auto bg-gradient-to-br from-brown-100 to-brown-200 rounded-xl shadow-xl border border-brown-300"
            >
              <table class="min-w-full divide-y divide-brown-300">
                <thead class="bg-brown-200/80">
                  <tr>
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >Name</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >🔧 Type</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >📝 Description</th
                    >
                    <th
                      class="px-4 py-3 text-left text-xs font-medium text-brown-900 uppercase tracking-wider"
                      >Actions</th
                    >
                  </tr>
                </thead>
                <tbody class="bg-brown-50/60 divide-y divide-brown-200">
                  {#each brewers as brewer}
                    <tr class="hover:bg-brown-100/60 transition-colors">
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{brewer.name}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-900"
                        >{brewer.brewer_type || "-"}</td
                      >
                      <td class="px-4 py-3 text-sm text-brown-700 italic max-w-xs"
                        >{brewer.description || "-"}</td
                      >
                      <td class="px-4 py-3 text-sm space-x-2">
                        <button
                          on:click={() => editBrewer(brewer)}
                          class="text-brown-700 hover:text-brown-900 font-medium"
                        >
                          Edit
                        </button>
                        <button
                          on:click={() => deleteBrewer(brewer.rkey)}
                          class="text-red-600 hover:text-red-800 font-medium"
                        >
                          Delete
                        </button>
                      </td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          {/if}
        {/if}
      </div>
    </div>
  {/if}
</div>

<!-- Modals -->
<Modal
  bind:isOpen={showBeanModal}
  title={editingBean ? "Edit Bean" : "Add Bean"}
  onSave={saveBean}
  onCancel={() => (showBeanModal = false)}
>
  <input
    type="text"
    bind:value={beanForm.name}
    placeholder="Name *"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  />
  <input
    type="text"
    bind:value={beanForm.origin}
    placeholder="Origin *"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  />
  <select
    bind:value={beanForm.roaster_rkey}
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  >
    <option value="">Select Roaster (Optional)</option>
    {#each roasters as roaster}
      <option value={roaster.rkey}>{roaster.name}</option>
    {/each}
  </select>
  <select
    bind:value={beanForm.roast_level}
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  >
    <option value="">Select Roast Level (Optional)</option>
    <option value="Ultra-Light">Ultra-Light</option>
    <option value="Light">Light</option>
    <option value="Medium-Light">Medium-Light</option>
    <option value="Medium">Medium</option>
    <option value="Medium-Dark">Medium-Dark</option>
    <option value="Dark">Dark</option>
  </select>
  <input
    type="text"
    bind:value={beanForm.process}
    placeholder="Process (e.g. Washed, Natural, Honey)"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  />
  <textarea
    bind:value={beanForm.description}
    placeholder="Description"
    rows="3"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  ></textarea>
</Modal>

<Modal
  bind:isOpen={showRoasterModal}
  title={editingRoaster ? "Edit Roaster" : "Add Roaster"}
  onSave={saveRoaster}
  onCancel={() => (showRoasterModal = false)}
>
  <input
    type="text"
    bind:value={roasterForm.name}
    placeholder="Name *"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  />
  <input
    type="text"
    bind:value={roasterForm.location}
    placeholder="Location"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  />
  <input
    type="url"
    bind:value={roasterForm.website}
    placeholder="Website"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  />
</Modal>

<Modal
  bind:isOpen={showGrinderModal}
  title={editingGrinder ? "Edit Grinder" : "Add Grinder"}
  onSave={saveGrinder}
  onCancel={() => (showGrinderModal = false)}
>
  <input
    type="text"
    bind:value={grinderForm.name}
    placeholder="Name *"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  />
  <select
    bind:value={grinderForm.grinder_type}
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  >
    <option value="">Select Grinder Type *</option>
    <option value="Hand">Hand</option>
    <option value="Electric">Electric</option>
    <option value="Portable Electric">Portable Electric</option>
  </select>
  <select
    bind:value={grinderForm.burr_type}
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  >
    <option value="">Select Burr Type (Optional)</option>
    <option value="Conical">Conical</option>
    <option value="Flat">Flat</option>
  </select>
  <textarea
    bind:value={grinderForm.notes}
    placeholder="Notes"
    rows="3"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  ></textarea>
</Modal>

<Modal
  bind:isOpen={showBrewerModal}
  title={editingBrewer ? "Edit Brewer" : "Add Brewer"}
  onSave={saveBrewer}
  onCancel={() => (showBrewerModal = false)}
>
  <input
    type="text"
    bind:value={brewerForm.name}
    placeholder="Name *"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  />
  <input
    type="text"
    bind:value={brewerForm.brewer_type}
    placeholder="Type (e.g., Pour-Over, Immersion, Espresso)"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  />
  <textarea
    bind:value={brewerForm.description}
    placeholder="Description"
    rows="3"
    class="w-full rounded-lg border-2 border-brown-300 bg-white shadow-sm py-2 px-3 focus:border-brown-600 focus:ring-brown-600"
  ></textarea>
</Modal>
