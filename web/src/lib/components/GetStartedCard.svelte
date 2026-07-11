<script lang="ts">
  import StationDrawer from "./StationDrawer.svelte";
  import type { OnboardingResponse, ReadinessStatus } from "../types/api";

  type Props = {
    onboarding: OnboardingResponse | null;
    mode?: "onboarding" | "library";
  };
  let { onboarding, mode = "onboarding" }: Props = $props();
  // Local override populated by refreshAfterSave(); falls back to the prop
  // so the component stays reactive to parent updates.
  let refreshedOnboarding = $state<OnboardingResponse | null>(null);
  let currentOnboarding = $derived(refreshedOnboarding ?? onboarding);
  let openDrawer = $state("");
  let refreshing = $state(false);
  let isLibrary = $derived(mode === "library");

  function stampState(done: boolean, current: boolean): string {
    return done ? "done" : current ? "current" : "todo";
  }
  function nextRequired(r: ReadinessStatus): string {
    if (!r.HasBrewer) return "brewer";
    if (!r.HasRoaster) return "roaster";
    if (!r.HasBean) return "bean";
    return "";
  }
  function remaining(r: ReadinessStatus): number {
    return Number(!r.HasBrewer) + Number(!r.HasRoaster) + Number(!r.HasBean);
  }
  function encouragement(r: ReadinessStatus): string {
    const missing = [
      !r.HasBrewer && "brewer",
      !r.HasRoaster && "roaster",
      !r.HasBean && "bean",
    ].filter(Boolean) as string[];
    if (missing.length === 3)
      return "Start anywhere, they all take less than a minute.";
    if (missing.length === 2)
      return `Add a ${missing[0]} and a ${missing[1]} to unlock your first brew.`;
    return `Just a ${missing[0]} left, you're almost there.`;
  }
  function itemLabels(kind: string): string[] {
    if (!currentOnboarding) return [];
    switch (kind) {
      case "brewer":
        return currentOnboarding.brewers.map((item) => item.name);
      case "roaster":
        return currentOnboarding.roasters.map((item) => item.name);
      case "bean":
        return currentOnboarding.beans.map((item) => item.name || item.origin);
      case "grinder":
        return currentOnboarding.grinders.map((item) => item.name);
      default:
        return [];
    }
  }
  function done(kind: string): boolean {
    if (!currentOnboarding) return false;
    if (kind === "brewer") return currentOnboarding.readiness.HasBrewer;
    if (kind === "roaster") return currentOnboarding.readiness.HasRoaster;
    if (kind === "bean") return currentOnboarding.readiness.HasBean;
    return currentOnboarding.grinders.length > 0;
  }
  function addLabel(kind: string, title: string): string {
    return itemLabels(kind).length
      ? "Add another"
      : `Add a${title === "Bean" ? "n" : ""} ${title.toLowerCase()}`;
  }
  async function refreshAfterSave() {
    openDrawer = "";
    refreshing = true;
    try {
      const response = await fetch("/api/onboarding", {
        headers: { Accept: "application/json" },
      });
      if (response.ok)
        refreshedOnboarding = (await response.json()) as OnboardingResponse;
    } finally {
      refreshing = false;
    }
  }
</script>

{#if currentOnboarding}
  <section class="onboarding-card" data-mode={mode} aria-busy={refreshing}>
    {#if !isLibrary}
      <div
        class="onboarding-progress"
        role="group"
        aria-label={`Setup progress: ${3 - remaining(currentOnboarding.readiness)} of 3 required items added`}
      >
        {#each ["brewer", "roaster", "bean"] as kind, index (kind)}
          <div
            class="stamp"
            data-state={stampState(
              done(kind),
              nextRequired(currentOnboarding.readiness) === kind,
            )}
            style={`--stamp-color: var(--type-${kind});`}
          >
            <span class="stamp-mark" aria-hidden="true">
              {#if done(kind)}
                <svg
                  class="stamp-glyph"
                  viewBox="0 0 20 20"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"><path d="M4 10.5l4 4 8-9"></path></svg
                >
              {:else}{index + 1}{/if}
            </span><span class="stamp-label">{kind}</span>
          </div>
          {#if index < 2}<span
              class="stamp-line"
              data-filled={done(kind) || undefined}
              style={done(kind)
                ? `--stamp-fill: var(--type-${kind});`
                : undefined}
            ></span>{/if}
        {/each}
      </div>
    {/if}

    <div class="stations">
      {#each [{ no: "01", kind: "brewer", title: "Brewer", hint: "Espresso machine, V60, French press, whatever you brew with.", required: true }, { no: "02", kind: "roaster", title: "Roaster", hint: "Who roasted the beans. Pick a favorite local roaster to start.", required: true }, { no: "03", kind: "bean", title: "Bean", hint: "Your current bag, single origin, blend, anything you're drinking.", required: true }, { no: "04", kind: "grinder", title: "Grinder", hint: "Optional, skip this if you brew with pre-ground beans.", required: false }] as station (station.kind)}
        <article
          class="station"
          data-no={station.no}
          data-kind={station.kind}
          data-optional={station.required ? undefined : "true"}
        >
          <header class="station-head">
            <span class="station-no">{station.no}</span>
            <h3 class="station-title">{station.title}</h3>
            {#if done(station.kind)}
              <span class="station-tag" data-tag="done"
                ><svg
                  width="12"
                  height="12"
                  viewBox="0 0 20 20"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  aria-hidden="true"><path d="M4 10.5l4 4 8-9"></path></svg
                >added</span
              >
            {:else if station.required}<span
                class="station-tag"
                data-tag="required">required</span
              >
            {:else}<span class="station-tag" data-tag="optional">optional</span
              >{/if}
          </header>
          <p class="station-hint">{station.hint}</p>
          {#if itemLabels(station.kind).length}
            <ul class="station-items">
              {#each itemLabels(station.kind).slice(0, 3) as item}<li
                  class="station-item"
                >
                  <span class="station-item-bullet" aria-hidden="true"
                  ></span><span>{item}</span>
                </li>{/each}
              {#if itemLabels(station.kind).length > 3}<li
                  class="station-item-more"
                >
                  + {itemLabels(station.kind).length - 3} more
                </li>{/if}
            </ul>
          {:else}
            <p class="station-empty">
              Nothing here yet, <strong
                >{station.kind === "brewer"
                  ? "add the one you brew with most"
                  : station.kind === "roaster"
                    ? "add a roaster you love"
                    : station.kind === "bean"
                      ? "add the bag you're on now"
                      : "skip if you brew pre-ground"}</strong
              >
            </p>
          {/if}
          <button
            type="button"
            class="station-add"
            onclick={() =>
              (openDrawer = openDrawer === station.kind ? "" : station.kind)}
            aria-expanded={openDrawer === station.kind}
            ><span class="station-add-icon" aria-hidden="true">+</span
            >{addLabel(station.kind, station.title)}</button
          >
        </article>
        <div class="station-drawer-row">
          {#if openDrawer === station.kind}<StationDrawer
              kind={station.kind}
              roasters={currentOnboarding.roasters}
              onclose={() => (openDrawer = "")}
              onsaved={refreshAfterSave}
            />{/if}
        </div>
      {/each}
    </div>

    {#if !isLibrary}
      {#if remaining(currentOnboarding.readiness) === 0}
        <div class="ready-panel" data-state="ready">
          <div class="ready-copy">
            <p class="ready-headline">You're ready to brew.</p>
            <p class="ready-sub">Your kit is set up. Pour something good.</p>
          </div>
          <a href="/brews/new" class="ready-cta"
            >Log your first brew <span
              class="ready-cta-arrow"
              aria-hidden="true">→</span
            ></a
          >
        </div>
      {:else}
        <div class="ready-panel" data-state="not-ready">
          <div class="ready-copy">
            <p class="ready-headline">
              <span class="ready-count"
                ><span class="ready-count-num"
                  >{remaining(currentOnboarding.readiness)}</span
                >{remaining(currentOnboarding.readiness) === 1
                  ? "step"
                  : "steps"} to go.</span
              >
            </p>
            <p class="ready-sub">
              {encouragement(currentOnboarding.readiness)}
            </p>
          </div>
        </div>
      {/if}
    {/if}
  </section>
{/if}
