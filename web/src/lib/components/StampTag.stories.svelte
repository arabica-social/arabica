<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import StampTag from "./StampTag.svelte";

  const { Story } = defineMeta({
    title: "Components/StampTag",
    component: StampTag,
    tags: ["autodocs"],
  });
</script>

<Story name="Inactive" args={{ label: "Quick notes", onclick: () => {} }} />
<Story
  name="Active"
  args={{ label: "Espresso", active: true, onclick: () => {} }}
/>
<Story
  name="Disabled"
  args={{ label: "Locked", disabled: true, onclick: () => {} }}
/>
