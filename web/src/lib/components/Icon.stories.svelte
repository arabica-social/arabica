<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import Icon, { iconPaths, type IconName } from "./Icon.svelte";

  const { Story } = defineMeta({
    title: "Components/Icon",
    component: Icon,
    tags: ["autodocs"],
  });

  const names = Object.keys(iconPaths) as IconName[];
</script>

<Story name="All icons">
  <div
    style="display: grid; grid-template-columns: repeat(auto-fill, minmax(7rem, 1fr)); gap: 0.75rem; padding: 1rem; max-width: 42rem;"
  >
    {#each names as name (name)}
      <div
        style="display: flex; flex-direction: column; align-items: center; gap: 0.4rem; padding: 0.6rem 0.3rem; border: 1px solid var(--card-border); border-radius: 0.5rem;"
      >
        <Icon {name} class="w-6 h-6" />
        <span style="font-size: 0.68rem; color: var(--text-muted);">{name}</span
        >
      </div>
    {/each}
  </div>
</Story>
<Story name="Sizes">
  <div
    style="display: flex; align-items: flex-end; gap: 1.25rem; padding: 1rem;"
  >
    <Icon name="coffee" class="w-3 h-3" />
    <Icon name="coffee" class="w-4 h-4" />
    <Icon name="coffee" class="w-6 h-6" />
    <Icon name="coffee" class="w-8 h-8" />
  </div>
</Story>
