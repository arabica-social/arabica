<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import LedgerHeader from "./LedgerHeader.svelte";

  const { Story } = defineMeta({
    title: "Components/LedgerHeader",
    component: LedgerHeader,
    tags: ["autodocs"],
  });
</script>

<Story name="My Coffee" args={{ title: "My Coffee" }} />
<Story
  name="With eyebrow and description"
  args={{
    title: "My Coffee",
    eyebrow: "Library",
    description:
      "Your beans, brewers, grinders, and roasters as they exist on your PDS.",
  }}
/>
<Story name="With back link" args={{ title: "Settings", showBack: true }} />
