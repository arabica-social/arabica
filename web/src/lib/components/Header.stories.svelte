<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import Header from "./Header.svelte";

  const { Story } = defineMeta({
    title: "Components/Header",
    component: Header,
    tags: ["autodocs"],
  });
</script>

<Story name="Visitor" />
<Story name="Signed in" globals={{ session: "in" }} />
