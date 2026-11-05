<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import FeedFilters from "./FeedFilters.svelte";

  const { Story } = defineMeta({
    title: "Components/FeedFilters",
    component: FeedFilters,
    tags: ["autodocs"],
  });

  const entityTabs = [
    { label: "All", value: "" },
    { label: "Brews", value: "brew" },
    { label: "Beans", value: "bean" },
    { label: "Recipes", value: "recipe" },
    { label: "Roasters", value: "roaster" },
    { label: "Grinders", value: "grinder" },
    { label: "Brewers", value: "brewer" },
  ];
</script>

<Story
  name="Community defaults"
  args={{
    typeFilter: "",
    sort: "",
    loading: false,
    onType: () => {},
    onSort: () => {},
  }}
/>
<Story
  name="Brews filtered"
  args={{
    typeFilter: "brew",
    sort: "popular",
    loading: false,
    onType: () => {},
    onSort: () => {},
    tabs: entityTabs,
  }}
/>
<Story
  name="Loading"
  args={{
    typeFilter: "",
    sort: "",
    loading: true,
    onType: () => {},
    onSort: () => {},
    tabs: entityTabs,
  }}
/>
