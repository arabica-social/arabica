<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import ActionBar from "./ActionBar.svelte";

  const { Story } = defineMeta({
    title: "Components/ActionBar",
    component: ActionBar,
    tags: ["autodocs"],
  });
</script>

<Story
  name="Signed out"
  args={{
    subjectURI: "at://did:plc:example/app.arabica.record.brew/abc",
    likeCount: 3,
    commentCount: 1,
    isAuthenticated: false,
    shareURL: "https://arabica.example/r/abc",
    shareTitle: "Morning pour",
    shareText: "Tasted like cocoa",
    viewURL: "/brews/abc",
  }}
/>
<Story
  name="Signed in owner"
  args={{
    subjectURI: "at://did:plc:example/app.arabica.record.brew/abc",
    likeCount: 3,
    commentCount: 1,
    isAuthenticated: true,
    isOwner: true,
    editURL: "/brews/abc/edit",
    deleteURL: "/api/brews/abc",
    deleteRedirect: "/",
    viewURL: "/brews/abc",
    shareURL: "https://arabica.example/r/abc",
    shareTitle: "Morning pour",
    shareText: "Tasted like cocoa",
  }}
/>
