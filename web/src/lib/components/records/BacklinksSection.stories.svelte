<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import type { BacklinksResult, BacklinkEntry } from "../../types/entity_view";
  import BacklinksSection from "./BacklinksSection.svelte";

  const { Story } = defineMeta({
    title: "Components/BacklinksSection",
    component: BacklinksSection,
    tags: ["autodocs"],
  });

  function entry(did: string, handle: string, name: string): BacklinkEntry {
    return {
      DID: did,
      Handle: handle,
      DisplayName: name,
      AvatarURL: "",
      RecordURI: `at://${did}/app.arabica.record.bean/r`,
      Collection: "app.arabica.record.bean",
      RKey: "r",
      CreatedAt: "2026-10-01T12:00:00Z",
      Title: name,
      Rating: 8,
      HasRating: true,
      ChainDepth: 1,
    };
  }

  const result: BacklinksResult = {
    LibraryEntries: [
      entry("did:plc:a", "mika.arabica.example", "Mika"),
      entry("did:plc:b", "sven.arabica.example", "Sven"),
      entry("did:plc:c", "oka.arabica.example", "Oka"),
    ],
    LibraryCount: 12,
    Usage: [
      {
        Key: "brew",
        Label: "brews",
        Entries: [
          entry("did:plc:a", "mika.arabica.example", "Mika"),
          entry("did:plc:d", "june.arabica.example", "June"),
        ],
        Count: 34,
        RatingAverage: 8.2,
        RatingCount: 21,
        Page: 1,
        PerPage: 50,
        HasPrev: false,
        HasNext: true,
      },
    ],
    UsageCount: 34,
    RatingAverage: 8.2,
    RatingCount: 21,
  };

  const libraryOnly: BacklinksResult = { ...result, Usage: [], UsageCount: 0 };
</script>

<Story
  name="Bean with usage"
  args={{
    result,
    detailURL: "/beans/mika.arabica.example/bean-rkey/backlinks",
  }}
/>
<Story
  name="Library only"
  args={{
    result: libraryOnly,
    detailURL: "/beans/mika.arabica.example/bean-rkey/backlinks",
  }}
/>
