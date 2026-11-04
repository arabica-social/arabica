<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import Avatar from "./Avatar.svelte";

  const { Story } = defineMeta({
    title: "Components/Avatar",
    component: Avatar,
    tags: ["autodocs"],
  });
</script>

<Story name="Placeholder sizes">
  <div style="display: flex; align-items: flex-end; gap: 1.5rem; padding: 1rem;">
    <Avatar displayName="Mika" size="sm" />
    <Avatar displayName="Patrick" size="md" />
    <Avatar displayName="Solveig" size="lg" />
  </div>
</Story>
<Story name="Long display name">
  <div style="padding: 1rem;">
    <Avatar displayName="Кофейный-энтузиаст" size="md" />
  </div>
</Story>
