<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import type { BacklinksResult, BacklinkEntry } from "../types/entity_view";
  import EntityViewLayout from "./EntityViewLayout.svelte";

  const { Story } = defineMeta({
    title: "Components/EntityViewLayout",
    component: EntityViewLayout,
    tags: ["autodocs"],
  });

  const libraryEntry: BacklinkEntry = {
    DID: "did:plc:a",
    Handle: "mika.arabica.example",
    DisplayName: "Mika",
    AvatarURL: "",
    RecordURI: "at://did:plc:a/app.arabica.record.roaster/r",
    Collection: "app.arabica.record.roaster",
    RKey: "r",
    CreatedAt: "2026-10-01T12:00:00Z",
    Title: "Mika",
    Rating: 0,
    HasRating: false,
    ChainDepth: 1,
  };

  const backlinks: BacklinksResult = {
    LibraryEntries: [libraryEntry],
    LibraryCount: 3,
    Usage: [],
    UsageCount: 0,
    RatingAverage: 0,
    RatingCount: 0,
  };
</script>

<Story
  name="Roaster record (owner)"
  args={{
    recordType: "roaster",
    title: "Karak Mornings",
    createdAt: "2026-09-01T12:00:00Z",
    authorDID: "did:plc:viewer",
    authorHandle: "viewer.arabica.example",
    authorDisplay: "Pat",
    backlinks,
    backlinksDetailURL:
      "/roasters/viewer.arabica.example/roaster-rkey/backlinks",
    subjectURI: "at://did:plc:viewer/app.arabica.record.roaster/roaster-rkey",
    subjectCID: "bafyrei-roaster",
    isLiked: true,
    likeCount: 2,
    commentCount: 0,
    shareURL:
      "https://arabica.example/roasters/viewer.arabica.example/roaster-rkey",
    isOwner: true,
    editURL: "/roasters/viewer.arabica.example/roaster-rkey/edit",
    deleteURL: "/api/roasters/roaster-rkey",
    deleteRedirect: "/my-coffee",
    isAuthenticated: true,
    currentUserDID: "did:plc:viewer",
    comments: [],
  }}
>
  <p style="margin: 0; color: var(--text-secondary); line-height: 1.6;">
    Small-batch roastery in Oslo focusing on washed Kenyan and Ethiopian lots.
  </p>
</Story>
