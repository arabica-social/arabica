<script lang="ts">
  import Avatar from "$lib/components/Avatar.svelte";
  import Icon from "$lib/components/Icon.svelte";
  import ActionBar from "$lib/components/ActionBar.svelte";
  import { displayHandle, safeAvatarURL } from "$lib/stores/session";
  import {
    pluralS,
    formatAvgRating,
    formatTemp,
    formatTime,
  } from "$lib/utils/format";
  import { pushToast } from "$lib/stores/toasts";
  import { goto } from "$app/navigation";
  import type { PageData } from "./$types";
  import type { ProfileResponse } from "$lib/types/api";
  import type { Bean, Brew } from "$lib/types/entity_view";

  let { data }: { data: PageData } = $props();

  type Tab = "brews" | "beans" | "equipment";
  let activeTab = $state<Tab>("brews");

  // svelte-ignore state_referenced_locally
  let profile = $state<ProfileResponse | null>(data.profile);

  // svelte-ignore state_referenced_locally
  $effect(() => {
    profile = data.profile;
  });

  function entityURI(did: string, collection: string, rkey: string): string {
    return `at://${did}/${collection}/${rkey}`;
  }

  function beanBrewCount(did: string, rkey: string): number {
    if (!profile) return 0;
    return (
      profile.bean_brew_counts[
        entityURI(did, "social.arabica.alpha.bean", rkey)
      ] ?? 0
    );
  }
  function beanAvgRating(did: string, rkey: string): number {
    if (!profile) return 0;
    return (
      profile.bean_avg_brew_ratings[
        entityURI(did, "social.arabica.alpha.bean", rkey)
      ] ?? 0
    );
  }
  function roasterBeanCount(did: string, rkey: string): number {
    if (!profile) return 0;
    return (
      profile.roaster_bean_counts[
        entityURI(did, "social.arabica.alpha.roaster", rkey)
      ] ?? 0
    );
  }
  function roasterAvgRating(did: string, rkey: string): number {
    if (!profile) return 0;
    return (
      profile.roaster_avg_brew_ratings[
        entityURI(did, "social.arabica.alpha.roaster", rkey)
      ] ?? 0
    );
  }
  function grinderBrewCount(did: string, rkey: string): number {
    if (!profile) return 0;
    return (
      profile.grinder_brew_counts[
        entityURI(did, "social.arabica.alpha.grinder", rkey)
      ] ?? 0
    );
  }
  function brewerBrewCount(did: string, rkey: string): number {
    if (!profile) return 0;
    return (
      profile.brewer_brew_counts[
        entityURI(did, "social.arabica.alpha.brewer", rkey)
      ] ?? 0
    );
  }

  let did = $derived(profile?.did ?? "");
  let isOwnProfile = $derived(profile?.is_own_profile ?? false);
  let isAuthenticated = $derived(data.isAuthenticated);

  function openBeans(): Bean[] {
    return (profile?.beans ?? []).filter((b) => !b.closed);
  }
  function closedBeans(): Bean[] {
    return (profile?.beans ?? []).filter((b) => b.closed);
  }

  let loadingMore = $state(false);
  async function loadMoreBrews() {
    if (!profile?.brews_has_more || loadingMore) return;
    loadingMore = true;
    try {
      const res = await fetch(
        `/api/profile/${data.actor}?brews_offset=${profile.brews_next_offset}&brews_limit=25`,
        {
          headers: { Accept: "application/json" },
        },
      );
      if (!res.ok) throw new Error("Failed");
      const next = (await res.json()) as ProfileResponse;
      if (profile) {
        profile = {
          ...profile,
          brews: [...profile.brews, ...next.brews],
          brews_has_more: next.brews_has_more,
          brews_next_offset: next.brews_next_offset,
        };
      }
    } catch {
      pushToast("Failed to load more");
    } finally {
      loadingMore = false;
    }
  }
</script>

<svelte:head>
  <title
    >{profile?.profile.display_name || profile?.profile.handle || "Profile"} - Arabica</title
  >
  {#if profile}
    <meta
      name="description"
      content={`${profile.profile.display_name || profile.profile.handle}'s coffee profile on Arabica`}
    />
  {/if}
</svelte:head>

{#if data.error && !profile}
  <div class="page-container-lg">
    <div class="card p-8 text-center">
      <div class="text-6xl mb-4 font-bold text-secondary">404</div>
      <h2 class="text-2xl font-bold text-primary mb-4">Page Not Found</h2>
      <p class="text-emphasis mb-6">{data.error}</p>
      <a href="/" class="btn-primary py-3 px-6">Back to Home</a>
    </div>
  </div>
{:else if profile}
  <div class="profile-ledger">
    <header class="profile-masthead">
      <Avatar
        avatarURL={safeAvatarURL(profile.profile.avatar)}
        displayName={profile.profile.display_name || profile.profile.handle}
        size="lg"
      />
      <div class="profile-identity">
        <p class="profile-label">Coffee ledger</p>
        <h1>
          {profile.profile.display_name ||
            displayHandle(profile.profile.handle)}
        </h1>
        <p class="profile-handle">@{displayHandle(profile.profile.handle)}</p>
      </div>
      {#if isOwnProfile}<a href="/settings" class="btn-secondary text-sm"
          >Settings</a
        >{/if}
    </header>

    <section class="profile-stats" aria-label="Coffee collection statistics">
      <div><strong>{profile.total_brews}</strong><span>Brews</span></div>
      <div><strong>{profile.beans.length}</strong><span>Beans</span></div>
      <div><strong>{profile.roasters.length}</strong><span>Roasters</span></div>
      <div><strong>{profile.grinders.length}</strong><span>Grinders</span></div>
      <div><strong>{profile.brewers.length}</strong><span>Brewers</span></div>
    </section>

    <div class="profile-grid">
      <aside class="profile-index" aria-label="Profile sections">
        <p class="profile-label">Index</p>
        <div role="tablist" aria-label="Profile collections">
          <button
            type="button"
            onclick={() => (activeTab = "brews")}
            class:active={activeTab === "brews"}
            role="tab"
            aria-selected={activeTab === "brews"}>Brews</button
          >
          <button
            type="button"
            onclick={() => (activeTab = "beans")}
            class:active={activeTab === "beans"}
            role="tab"
            aria-selected={activeTab === "beans"}>Beans</button
          >
          <button
            type="button"
            onclick={() => (activeTab = "equipment")}
            class:active={activeTab === "equipment"}
            role="tab"
            aria-selected={activeTab === "equipment"}>Gear</button
          >
        </div>
      </aside>
      <main class="profile-content">
        <!-- Brews tab -->
        {#if activeTab === "brews"}
          <div class="space-y-4">
            {#if profile.brews.length === 0}
              <div class="card card-inner text-center py-8">
                {#if isOwnProfile}
                  <p class="text-secondary mb-4">
                    No brews recorded yet. Add the cup you're drinking now.
                  </p>
                  <a href="/brews/new" class="btn-primary"
                    >Add your first brew</a
                  >
                {:else}
                  <p class="text-secondary">No brews yet.</p>
                {/if}
              </div>
            {:else}
              {#each profile.brews as brew (brew.rkey)}
                <div class="feed-card feed-card-brew">
                  <a href={`/brews/${did}/${brew.rkey}`} class="block">
                    <div class="flex items-center justify-between mb-2">
                      <span class="text-sm text-muted"
                        >{new Date(brew.created_at).toLocaleDateString(
                          undefined,
                          { month: "short", day: "numeric", year: "numeric" },
                        )}</span
                      >
                      {#if brew.rating > 0}
                        <span class="badge-rating flex items-center gap-1">
                          <Icon name="star" class="w-3 h-3 text-amber-500" />
                          {brew.rating}/10
                        </span>
                      {/if}
                    </div>
                    {#if brew.bean}
                      <div class="font-bold text-primary">
                        {brew.bean.name || brew.bean.origin}
                      </div>
                      {#if brew.bean.roaster?.name}
                        <div class="text-sm text-muted flex items-center gap-1">
                          <Icon name="store" class="w-3 h-3" />
                          {brew.bean.roaster.name}
                        </div>
                      {/if}
                      <div
                        class="text-xs text-faint mt-1 flex flex-wrap gap-x-2 gap-y-1"
                      >
                        {#if brew.bean.origin}<span
                            class="inline-flex items-center gap-1"
                            ><Icon name="mapPin" class="w-3 h-3" />{brew.bean
                              .origin}</span
                          >{/if}
                        {#if brew.bean.roast_level}<span
                            class="inline-flex items-center gap-1"
                            ><Icon name="flame" class="w-3 h-3" />{brew.bean
                              .roast_level}</span
                          >{/if}
                        {#if brew.coffee_amount > 0}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="scale"
                              class="w-3 h-3"
                            />{brew.coffee_amount}g</span
                          >{/if}
                      </div>
                    {/if}
                    {#if brew.brewer_obj?.name || brew.method}
                      <div class="mb-2">
                        <span class="text-meta">Brewer:</span>
                        <span class="text-sm font-semibold text-primary"
                          >{brew.brewer_obj?.name ?? brew.method}</span
                        >
                      </div>
                    {/if}
                    <div
                      class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs text-emphasis"
                    >
                      {#if brew.grinder_obj?.name}
                        <div>
                          <span class="text-label">Grinder:</span>
                          {brew.grinder_obj.name}{#if brew.grind_size}
                            ({brew.grind_size}){/if}
                        </div>
                      {:else if brew.grind_size}
                        <div>
                          <span class="text-label">Grind:</span>
                          {brew.grind_size}
                        </div>
                      {/if}
                      {#if brew.water_amount > 0}<div>
                          <span class="text-label">Water:</span>
                          {brew.water_amount}g
                        </div>{/if}
                      {#if brew.temperature > 0}<div>
                          <span class="text-label">Temp:</span>
                          {formatTemp(brew.temperature)}
                        </div>{/if}
                      {#if brew.time_seconds > 0}<div>
                          <span class="text-label">Time:</span>
                          {formatTime(brew.time_seconds)}
                        </div>{/if}
                    </div>
                    {#if brew.tasting_notes}
                      <div class="text-sm text-secondary mt-1 line-clamp-2">
                        {brew.tasting_notes}
                      </div>
                    {/if}
                  </a>
                  {#if data.isAuthenticated}
                    <ActionBar
                      subjectURI={`at://${did}/social.arabica.alpha.brew/${brew.rkey}`}
                      subjectCID={profile.brew_cids[brew.rkey] ?? ""}
                      isLiked={profile.brew_liked_by_user[brew.rkey] ?? false}
                      likeCount={profile.brew_like_counts[brew.rkey] ?? 0}
                      commentCount={profile.brew_comment_counts[brew.rkey] ?? 0}
                      shareURL={`/brews/${did}/${brew.rkey}`}
                      shareTitle={brew.bean
                        ? brew.bean.name || brew.bean.origin
                        : "Brew"}
                      shareText="Check out this brew on Arabica"
                      isOwner={isOwnProfile}
                      deleteURL={isOwnProfile ? `/api/brews/${brew.rkey}` : ""}
                      deleteRedirect="/profile/{data.actor}"
                      viewURL={`/brews/${did}/${brew.rkey}`}
                      isAuthenticated={data.isAuthenticated}
                    />
                  {/if}
                </div>
              {/each}
              {#if profile.brews_has_more}
                <div class="text-center py-4">
                  <button
                    type="button"
                    class="btn-secondary px-6 py-2"
                    onclick={loadMoreBrews}
                    disabled={loadingMore}
                  >
                    {loadingMore ? "Loading..." : "Load More"}
                  </button>
                </div>
              {/if}
            {/if}
          </div>
        {/if}

        <!-- Beans tab -->
        {#if activeTab === "beans"}
          <div class="space-y-6">
            <!-- Open bags -->
            <div>
              <h4 class="text-lg font-semibold text-primary mb-3">Open bags</h4>
              {#if openBeans().length === 0}
                <div class="card card-inner text-center py-6">
                  <p class="text-secondary">
                    {isOwnProfile ? "No open bags yet." : "No open bags yet."}
                  </p>
                </div>
              {:else}
                <div
                  class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
                >
                  {#each openBeans() as bean (bean.rkey)}
                    <div class="feed-card feed-card-bean">
                      <a
                        href={`/beans/${did}/${bean.rkey}`}
                        class="font-bold text-primary hover:underline"
                        >{bean.name || bean.origin}</a
                      >
                      {#if bean.roaster?.name}
                        <div class="text-sm text-muted flex items-center gap-1">
                          <Icon name="store" class="w-3 h-3" />
                          {bean.roaster.name}
                        </div>
                      {/if}
                      <div
                        class="text-xs text-faint mt-1 flex flex-wrap gap-x-2 gap-y-1"
                      >
                        {#if bean.origin}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="mapPin"
                              class="w-3 h-3"
                            />{bean.origin}</span
                          >{/if}
                        {#if bean.roast_level}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="flame"
                              class="w-3 h-3"
                            />{bean.roast_level}</span
                          >{/if}
                        {#if bean.variety}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="leaf"
                              class="w-3 h-3"
                            />{bean.variety}</span
                          >{/if}
                        {#if bean.process}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="sprout"
                              class="w-3 h-3"
                            />{bean.process}</span
                          >{/if}
                      </div>
                      {#if beanBrewCount(did, bean.rkey) > 0 || beanAvgRating(did, bean.rkey) > 0}
                        <div
                          class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60 flex gap-3"
                        >
                          {#if beanBrewCount(did, bean.rkey) > 0}<span
                              class="inline-flex items-center gap-1"
                              ><Icon
                                name="coffee"
                                class="w-3 h-3"
                              />{beanBrewCount(did, bean.rkey)} brew{pluralS(
                                beanBrewCount(did, bean.rkey),
                              )}</span
                            >{/if}
                          {#if beanAvgRating(did, bean.rkey) > 0}<span
                              class="inline-flex items-center gap-1"
                              ><Icon
                                name="star"
                                class="w-3 h-3 text-amber-500"
                              />{formatAvgRating(beanAvgRating(did, bean.rkey))} avg</span
                            >{/if}
                        </div>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
            <!-- Roasters -->
            <div>
              <h4 class="text-lg font-semibold text-primary mb-3">Roasters</h4>
              {#if profile.roasters.length === 0}
                <div class="card card-inner text-center py-6">
                  <p class="text-secondary">No roasters yet.</p>
                </div>
              {:else}
                <div
                  class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
                >
                  {#each profile.roasters as roaster (roaster.rkey)}
                    <div class="feed-card feed-card-roaster">
                      <a
                        href={`/roasters/${did}/${roaster.rkey}`}
                        class="font-bold text-primary hover:underline"
                        >{roaster.name}</a
                      >
                      {#if roaster.location}
                        <div
                          class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1"
                        >
                          <span class="inline-flex items-center gap-1"
                            ><Icon
                              name="mapPin"
                              class="w-3 h-3"
                            />{roaster.location}</span
                          >
                        </div>
                      {/if}
                      {#if roasterBeanCount(did, roaster.rkey) > 0 || roasterAvgRating(did, roaster.rkey) > 0}
                        <div
                          class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60 flex gap-3"
                        >
                          {#if roasterBeanCount(did, roaster.rkey) > 0}<span
                              class="inline-flex items-center gap-1"
                              ><Icon
                                name="leaf"
                                class="w-3 h-3"
                              />{roasterBeanCount(did, roaster.rkey)} bean{pluralS(
                                roasterBeanCount(did, roaster.rkey),
                              )}</span
                            >{/if}
                          {#if roasterAvgRating(did, roaster.rkey) > 0}<span
                              class="inline-flex items-center gap-1"
                              ><Icon
                                name="star"
                                class="w-3 h-3 text-amber-500"
                              />{formatAvgRating(
                                roasterAvgRating(did, roaster.rkey),
                              )} avg</span
                            >{/if}
                        </div>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
            <!-- Closed bags -->
            {#if closedBeans().length > 0}
              <div>
                <h4 class="text-lg font-semibold text-primary mb-3">
                  Finished bags
                </h4>
                <div
                  class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
                >
                  {#each closedBeans() as bean (bean.rkey)}
                    <div class="feed-card feed-card-bean opacity-75">
                      <a
                        href={`/beans/${did}/${bean.rkey}`}
                        class="font-bold text-primary hover:underline"
                        >{bean.name || bean.origin}</a
                      >
                      <span class="text-xs text-faint">Closed</span>
                      {#if bean.roaster?.name}
                        <div class="text-sm text-muted flex items-center gap-1">
                          <Icon name="store" class="w-3 h-3" />
                          {bean.roaster.name}
                        </div>
                      {/if}
                      <div
                        class="text-xs text-faint mt-1 flex flex-wrap gap-x-2 gap-y-1"
                      >
                        {#if bean.origin}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="mapPin"
                              class="w-3 h-3"
                            />{bean.origin}</span
                          >{/if}
                        {#if bean.roast_level}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="flame"
                              class="w-3 h-3"
                            />{bean.roast_level}</span
                          >{/if}
                        {#if bean.variety}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="leaf"
                              class="w-3 h-3"
                            />{bean.variety}</span
                          >{/if}
                        {#if bean.process}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="sprout"
                              class="w-3 h-3"
                            />{bean.process}</span
                          >{/if}
                      </div>
                      {#if beanBrewCount(did, bean.rkey) > 0 || beanAvgRating(did, bean.rkey) > 0}
                        <div
                          class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60 flex gap-3"
                        >
                          {#if beanBrewCount(did, bean.rkey) > 0}<span
                              class="inline-flex items-center gap-1"
                              ><Icon
                                name="coffee"
                                class="w-3 h-3"
                              />{beanBrewCount(did, bean.rkey)} brew{pluralS(
                                beanBrewCount(did, bean.rkey),
                              )}</span
                            >{/if}
                          {#if beanAvgRating(did, bean.rkey) > 0}<span
                              class="inline-flex items-center gap-1"
                              ><Icon
                                name="star"
                                class="w-3 h-3 text-amber-500"
                              />{formatAvgRating(beanAvgRating(did, bean.rkey))} avg</span
                            >{/if}
                        </div>
                      {/if}
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        {/if}

        <!-- Equipment tab -->
        {#if activeTab === "equipment"}
          <div class="space-y-6">
            <!-- Grinders -->
            <div>
              <h4 class="text-lg font-semibold text-primary mb-3">Grinders</h4>
              {#if profile.grinders.length === 0}
                <div class="card card-inner text-center py-6">
                  <p class="text-secondary">No grinders yet.</p>
                </div>
              {:else}
                <div
                  class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
                >
                  {#each profile.grinders as grinder (grinder.rkey)}
                    <div class="feed-card feed-card-grinder">
                      <a
                        href={`/grinders/${did}/${grinder.rkey}`}
                        class="font-bold text-primary hover:underline"
                        >{grinder.name}</a
                      >
                      <div
                        class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1"
                      >
                        {#if grinder.grinder_type}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="tag"
                              class="w-3 h-3"
                            />{grinder.grinder_type}</span
                          >{/if}
                        {#if grinder.burr_type}<span
                            class="inline-flex items-center gap-1"
                            ><Icon
                              name="disc"
                              class="w-3 h-3"
                            />{grinder.burr_type}</span
                          >{/if}
                      </div>
                      {#if grinderBrewCount(did, grinder.rkey) > 0}
                        <div
                          class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60"
                        >
                          <span class="inline-flex items-center gap-1"
                            ><Icon
                              name="coffee"
                              class="w-3 h-3"
                            />{grinderBrewCount(did, grinder.rkey)} brew{pluralS(
                              grinderBrewCount(did, grinder.rkey),
                            )}</span
                          >
                        </div>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
            <div>
              <h4 class="text-lg font-semibold text-primary mb-3">Brewers</h4>
              {#if profile.brewers.length === 0}
                <div class="card card-inner text-center py-6">
                  <p class="text-secondary">No brewers yet.</p>
                </div>
              {:else}
                <div
                  class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4"
                >
                  {#each profile.brewers as brewer (brewer.rkey)}
                    <div class="feed-card feed-card-brewer">
                      <a
                        href={`/brewers/${did}/${brewer.rkey}`}
                        class="font-bold text-primary hover:underline"
                        >{brewer.name}</a
                      >
                      {#if brewer.brewer_type}
                        <div
                          class="text-xs text-muted mt-1 flex flex-wrap gap-x-2 gap-y-1"
                        >
                          <span class="inline-flex items-center gap-1"
                            ><Icon
                              name="brewer"
                              class="w-3 h-3"
                            />{brewer.brewer_type}</span
                          >
                        </div>
                      {/if}
                      {#if brewerBrewCount(did, brewer.rkey) > 0}
                        <div
                          class="text-xs text-faint pt-2 mt-2 border-t border-brown-200/60"
                        >
                          <span class="inline-flex items-center gap-1"
                            ><Icon
                              name="coffee"
                              class="w-3 h-3"
                            />{brewerBrewCount(did, brewer.rkey)} brew{pluralS(
                              brewerBrewCount(did, brewer.rkey),
                            )}</span
                          >
                        </div>
                      {/if}
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          </div>
        {/if}
      </main>
    </div>
  </div>
{/if}

<style>
  .profile-ledger {
    width: 100%;
    max-width: 76rem;
    margin-inline: auto;
    padding-inline: clamp(0.5rem, 2.2vw, 2rem);
  }
  .profile-masthead {
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    align-items: center;
    gap: 1rem;
    padding: 0.75rem 0 1.5rem;
    border-bottom: 1px solid var(--card-border);
  }
  .profile-label {
    margin: 0 0 0.35rem;
    color: var(--text-muted);
    font-size: 0.65rem;
    font-weight: 700;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }
  .profile-identity h1 {
    margin: 0;
    color: var(--text-primary);
    font-family: var(--font-display);
    font-size: clamp(2rem, 4vw, 3rem);
    font-weight: 600;
    line-height: 1;
    letter-spacing: -0.025em;
  }
  .profile-handle {
    margin: 0.4rem 0 0;
    color: var(--text-emphasis);
    font-size: 0.85rem;
  }
  .profile-stats {
    display: grid;
    grid-template-columns: repeat(5, minmax(7rem, 1fr));
    overflow-x: auto;
    margin-bottom: 1.75rem;
    border-bottom: 1px solid var(--card-border);
  }
  .profile-stats div {
    display: flex;
    align-items: baseline;
    gap: 0.45rem;
    min-width: 7rem;
    padding: 0.9rem 1rem 0.9rem 0;
    border-right: 1px solid var(--card-border);
  }
  .profile-stats div:last-child {
    border-right: 0;
  }
  .profile-stats strong {
    color: var(--text-primary);
    font-family: var(--font-display);
    font-size: 1.45rem;
    margin-left: 0.5rem;
    font-variant-numeric: tabular-nums;
  }
  .profile-stats span {
    color: var(--text-muted);
    font-size: 0.65rem;
    letter-spacing: 0.08em;
    text-transform: uppercase;
  }
  .profile-grid {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1.5rem;
  }
  .profile-index {
    min-width: 0;
  }
  .profile-index [role="tablist"] {
    display: flex;
    overflow-x: auto;
    border-block: 1px solid var(--card-border);
  }
  .profile-index button {
    min-width: max-content;
    min-height: 2.75rem;
    padding: 0.6rem 0.85rem;
    color: var(--text-muted);
    border-bottom: 2px solid transparent;
    font-size: 0.78rem;
  }
  .profile-index button.active {
    color: var(--text-primary);
    border-bottom-color: var(--text-secondary);
    font-weight: 600;
  }
  .profile-content {
    min-width: 0;
  }
  @media (min-width: 860px) {
    .profile-grid {
      grid-template-columns: minmax(9rem, 0.55fr) minmax(0, 2.8fr);
      gap: clamp(1.5rem, 3vw, 3rem);
      align-items: start;
    }
    .profile-index {
      position: sticky;
      top: 5rem;
    }
    .profile-index [role="tablist"] {
      display: block;
      overflow: visible;
      border-block: 0;
      border-top: 2px solid var(--text-secondary);
    }
    .profile-index button {
      width: 100%;
      display: block;
      text-align: left;
      border-bottom: 1px dotted var(--card-border);
      border-left: 2px solid transparent;
    }
    .profile-index button.active {
      border-bottom-color: var(--card-border);
      border-left-color: var(--text-secondary);
    }
  }
  @media (max-width: 560px) {
    .profile-masthead {
      grid-template-columns: auto 1fr;
    }
    .profile-masthead > :global(a) {
      grid-column: 1/-1;
      text-align: center;
    }
  }
</style>
