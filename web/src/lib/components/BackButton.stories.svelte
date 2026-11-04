<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import BackButton from "./BackButton.svelte";

  const { Story } = defineMeta({
    title: "Components/BackButton",
    component: BackButton,
    tags: ["autodocs"],
  });
</script>

<Story name="Default" />
