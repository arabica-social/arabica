<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import HomeEntry from "./HomeEntry.svelte";

  const { Story } = defineMeta({
    title: "AI/HomeEntry",
    component: HomeEntry,
    tags: ["autodocs"],
  });
</script>

<Story name="Ready" args={{ accountDid: "did:plc:storybook-viewer" }} />
