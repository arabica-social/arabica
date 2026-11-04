<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import Footer from "./Footer.svelte";

  const { Story } = defineMeta({
    title: "Components/Footer",
    component: Footer,
    tags: ["autodocs"],
  });
</script>

<Story name="Default" />
