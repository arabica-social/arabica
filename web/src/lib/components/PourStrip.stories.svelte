<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import type { Pour } from "../types/entity_view";
  import PourStrip from "./PourStrip.svelte";

  const { Story } = defineMeta({
    title: "Components/PourStrip",
    component: PourStrip,
    tags: ["autodocs"],
  });

  const pours: Pour[] = [
    { pour_number: 1, water_amount: 50, time_seconds: 45, created_at: "" },
    { pour_number: 2, water_amount: 100, time_seconds: 20, created_at: "" },
    { pour_number: 3, water_amount: 100, time_seconds: 20, created_at: "" },
    { pour_number: 4, water_amount: 50, time_seconds: 15, created_at: "" },
  ];
</script>

<Story name="Four pours" args={{ pours }} />
<Story
  name="Zero-time pours"
  args={{ pours: pours.map((pour) => ({ ...pour, time_seconds: 0 })) }}
/>
<Story name="Empty" args={{ pours: [] }} />
