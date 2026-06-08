<script lang="ts">
  import { onMount } from "svelte";

  type TasteAxis = {
    id: string;
    label: string;
    value: number;
    evidence?: string;
  };

  let { target }: { target: HTMLElement } = $props();
  let axes = $state<TasteAxis[]>([]);

  const size = 360;
  const center = size / 2;
  const outerRadius = 122;
  const innerRadius = 38;
  const labelRadius = 158;

  let hasSignals = $derived(axes.some((axis) => axis.value > 0));
  let points = $derived(
    axes.map((axis, index) => axisPoint(axis, index, outerRadius)),
  );
  let polygonPoints = $derived(
    points.map((point) => `${point.x},${point.y}`).join(" "),
  );

  function angleFor(index: number) {
    return -90 + (360 / Math.max(axes.length, 1)) * index;
  }

  function polarPoint(radius: number, angleDegrees: number) {
    const angle = (angleDegrees * Math.PI) / 180;
    return {
      x: center + radius * Math.cos(angle),
      y: center + radius * Math.sin(angle),
    };
  }

  function axisPoint(axis: TasteAxis, index: number, radius: number) {
    const clamped = Math.max(0, Math.min(axis.value, 100));
    const scaled = innerRadius + ((radius - innerRadius) * clamped) / 100;
    return polarPoint(scaled, angleFor(index));
  }

  function labelPoint(index: number) {
    return polarPoint(labelRadius, angleFor(index));
  }

  function textAnchor(index: number) {
    const point = labelPoint(index);
    if (Math.abs(point.x - center) < 12) return "middle";
    return point.x > center ? "start" : "end";
  }

  function readAxesData() {
    const data = document.getElementById(
      "taste-profile-data",
    ) as HTMLElement | null;
    if (!data?.dataset.profile) {
      axes = [];
      return;
    }

    try {
      const parsed = JSON.parse(data.dataset.profile) as TasteAxis[];
      axes = parsed
        .filter((axis) => axis.id && axis.label)
        .map((axis) => ({
          id: axis.id,
          label: axis.label,
          value: Math.max(0, Math.min(Number(axis.value) || 0, 100)),
          evidence: axis.evidence || "",
        }));
    } catch (error) {
      console.warn("taste profile: failed to parse profile data:", error);
      axes = [];
    }
  }

  onMount(() => {
    const handleSwap = (event: Event) => {
      const detail = (event as CustomEvent<{ target?: Element }>).detail;
      if (
        (detail?.target as HTMLElement | undefined)?.id === "profile-content"
      ) {
        readAxesData();
      }
    };

    document.body.addEventListener("htmx:afterSwap", handleSwap);
    readAxesData();
    return () => {
      document.body.removeEventListener("htmx:afterSwap", handleSwap);
    };
  });
</script>

{#if axes.length > 0 && hasSignals}
  <section class="taste-profile" aria-labelledby="taste-profile-title">
    <div class="taste-profile-copy">
      <p class="taste-profile-kicker">Taste profile</p>
      <h2 id="taste-profile-title">What keeps showing up</h2>
    </div>
    <div class="taste-profile-chart">
      <svg
        viewBox={`0 0 ${size} ${size}`}
        role="img"
        aria-labelledby="taste-profile-title taste-profile-desc"
      >
        <desc id="taste-profile-desc">
          A radial taste profile where labels closer to the outside indicate
          stronger signals.
        </desc>
        <g class="taste-rings">
          <circle cx={center} cy={center} r={innerRadius} />
          <circle cx={center} cy={center} r={(innerRadius + outerRadius) / 2} />
          <circle cx={center} cy={center} r={outerRadius} />
        </g>
        {#each axes as axis, index}
          {@const label = labelPoint(index)}
          {@const end = polarPoint(outerRadius, angleFor(index))}
          <line
            class="taste-spoke"
            x1={center}
            y1={center}
            x2={end.x}
            y2={end.y}
          />
          <text
            class="taste-label"
            x={label.x}
            y={label.y}
            text-anchor={textAnchor(index)}
            dominant-baseline="middle"
          >
            {axis.label}
          </text>
        {/each}
        <polygon class="taste-area" points={polygonPoints} />
        <polyline
          class="taste-line"
          points={`${polygonPoints} ${points[0]?.x || center},${points[0]?.y || center}`}
        />
        {#each points as point, index}
          <circle class="taste-point" cx={point.x} cy={point.y} r="4.5">
            <title
              >{axes[index].label}: {axes[index].value}/100, {axes[index]
                .evidence}</title
            >
          </circle>
        {/each}
      </svg>
    </div>
    <div class="taste-profile-list" aria-hidden="true">
      {#each axes as axis}
        <div class="taste-profile-row">
          <span>{axis.label}</span>
          <span>{axis.evidence}</span>
        </div>
      {/each}
    </div>
  </section>
{/if}
