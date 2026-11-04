<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import SendButton from "./SendButton.svelte";

  const { Story } = defineMeta({
    title: "AI/SendButton",
    component: SendButton,
    tags: ["autodocs"],
  });
</script>

<Story name="Ready" />
<Story name="Disabled" args={{ disabled: true }} />
