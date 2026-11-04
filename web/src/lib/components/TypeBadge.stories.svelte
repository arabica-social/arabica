<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import TypeBadge from "./TypeBadge.svelte";

  const { Story } = defineMeta({
    title: "Components/TypeBadge",
    component: TypeBadge,
    tags: ["autodocs"],
  });
</script>

<Story name="Brew" args={{ recordType: "brew" }} />
<Story name="Bean" args={{ recordType: "bean" }} />
<Story name="Recipe" args={{ recordType: "recipe" }} />
<Story name="Unknown type" args={{ recordType: "mystery" }} />
