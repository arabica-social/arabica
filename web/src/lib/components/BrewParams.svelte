<script lang="ts">
  import { formatTemp, formatTime } from "../utils/format";
  import type { Brew } from "../types/entity_view";

  type Props = { brew: Brew; showPourCount?: boolean };
  let { brew, showPourCount = false }: Props = $props();

  let ep = $derived(brew.espresso_params ?? null);
  let pp = $derived(brew.pourover_params ?? null);
  let brewerName = $derived(brew.brewer_obj?.name ?? brew.method ?? "");
  let hasParams = $derived(
    (showPourCount && !!brew.pours?.length) ||
      brewerName !== "" ||
      !!brew.grinder_obj?.name ||
      !!brew.grind_size ||
      brew.coffee_amount > 0 ||
      brew.water_amount > 0 ||
      brew.temperature > 0 ||
      brew.time_seconds > 0 ||
      (ep !== null &&
        (ep.yield_weight > 0 ||
          ep.pressure > 0 ||
          ep.pre_infusion_seconds > 0)) ||
      (pp !== null &&
        (pp.bloom_water > 0 ||
          pp.bloom_seconds > 0 ||
          pp.drawdown_seconds > 0 ||
          pp.bypass_water > 0 ||
          !!pp.filter)),
  );

  function bloomText(p: NonNullable<Brew["pourover_params"]>): string {
    const parts: string[] = [];
    if (p.bloom_water > 0) parts.push(`${p.bloom_water}g`);
    if (p.bloom_seconds > 0) parts.push(`${p.bloom_seconds}s`);
    return parts.join(" for ");
  }
</script>

{#if hasParams}
  <div class="brew-params">
    {#if brewerName}
      <div>
        <span class="text-label">Brewer</span><span class="param-value"
          >{brewerName}</span
        >
      </div>
    {/if}
    {#if brew.grinder_obj?.name}
      <div>
        <span class="text-label">Grinder</span>
        <span class="param-value"
          >{brew.grinder_obj.name}{#if brew.grind_size}
            ({brew.grind_size}){/if}</span
        >
      </div>
    {:else if brew.grind_size}
      <div>
        <span class="text-label">Grind</span><span class="param-value"
          >{brew.grind_size}</span
        >
      </div>
    {/if}
    {#if brew.coffee_amount > 0}
      <div>
        <span class="text-label">Dose</span><span class="param-value"
          >{brew.coffee_amount}g</span
        >
      </div>
    {/if}
    {#if brew.water_amount > 0}
      <div>
        <span class="text-label">Water</span><span class="param-value"
          >{brew.water_amount}g</span
        >
      </div>
    {/if}
    {#if brew.temperature > 0}
      <div>
        <span class="text-label">Temp</span><span class="param-value"
          >{formatTemp(brew.temperature)}</span
        >
      </div>
    {/if}
    {#if brew.time_seconds > 0}
      <div>
        <span class="text-label">Time</span><span class="param-value"
          >{formatTime(brew.time_seconds)}</span
        >
      </div>
    {/if}
    {#if ep && ep.yield_weight > 0}
      <div>
        <span class="text-label">Yield</span><span class="param-value"
          >{ep.yield_weight.toFixed(1)}g</span
        >
      </div>
    {/if}
    {#if ep && ep.pressure > 0}
      <div>
        <span class="text-label">Pressure</span><span class="param-value"
          >{ep.pressure.toFixed(1)} bar</span
        >
      </div>
    {/if}
    {#if ep && ep.pre_infusion_seconds > 0}
      <div>
        <span class="text-label">Pre-infusion</span><span class="param-value"
          >{ep.pre_infusion_seconds}s</span
        >
      </div>
    {/if}
    {#if pp && (pp.bloom_water > 0 || pp.bloom_seconds > 0)}
      <div>
        <span class="text-label">Bloom</span><span class="param-value"
          >{bloomText(pp)}</span
        >
      </div>
    {/if}
    {#if pp && pp.drawdown_seconds > 0}
      <div>
        <span class="text-label">Drawdown</span><span class="param-value"
          >{pp.drawdown_seconds}s</span
        >
      </div>
    {/if}
    {#if pp && pp.bypass_water > 0}
      <div>
        <span class="text-label">Bypass</span><span class="param-value"
          >{pp.bypass_water}g</span
        >
      </div>
    {/if}
    {#if showPourCount && brew.pours?.length}
      <div>
        <span class="text-label">Pours</span><span class="param-value"
          >{brew.pours.length}</span
        >
      </div>
    {/if}
    {#if pp && pp.filter}
      <div>
        <span class="text-label">Filter</span><span class="param-value"
          >{pp.filter}</span
        >
      </div>
    {/if}
  </div>
{/if}
