<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import ScrollTopButton from "./ScrollTopButton.svelte";

  const { Story } = defineMeta({
    title: "Components/ScrollTopButton",
    component: ScrollTopButton,
    tags: ["autodocs"],
  });
</script>

<Story name="Tall page (scroll to reveal)">
  <div
    style="height: 180vh; padding: 1rem; color: var(--text-muted); font-size: 0.85rem;"
  >
    Scroll down 400px and the floating button appears.
  </div>
  <ScrollTopButton />
</Story>
