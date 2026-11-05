<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import type { BacklinksResult, BacklinkEntry } from "../../types/entity_view";
  import BacklinksView from "./BacklinksView.svelte";

  const { Story } = defineMeta({
    title: "Components/BacklinksView",
    component: BacklinksView,
    tags: ["autodocs"],
  });

  function entry(
    did: string,
    handle: string,
    name: string,
    title: string,
    collection: string,
  ): BacklinkEntry {
    return {
      DID: did,
      Handle: handle,
      DisplayName: name,
      AvatarURL: "",
      RecordURI: `at://${did}/${collection}/r`,
      Collection: collection,
      RKey: "r",
      CreatedAt: "2026-10-20T18:00:00Z",
      Title: title,
      Rating: 9,
      HasRating: true,
      ChainDepth: 1,
    };
  }

  const result: BacklinksResult = {
    LibraryEntries: [
      entry(
        "did:plc:a",
        "mika.arabica.example",
        "Mika",
        "Mika's library",
        "app.arabica.record.bean",
      ),
      entry(
        "did:plc:b",
        "sven.arabica.example",
        "Sven",
        "Sven's library",
        "app.arabica.record.bean",
      ),
    ],
    LibraryCount: 2,
    Usage: [
      {
        Key: "brew",
        Label: "brews",
        Entries: [
          entry(
            "did:plc:a",
            "mika.arabica.example",
            "Mika",
            "Saturday espresso",
            "app.arabica.record.brew",
          ),
        ],
        Count: 5,
        RatingAverage: 8.5,
        RatingCount: 3,
        Page: 1,
        PerPage: 50,
        HasPrev: false,
        HasNext: false,
      },
    ],
    UsageCount: 5,
    RatingAverage: 8.5,
    RatingCount: 3,
  };
</script>

<Story
  name="Bean detail"
  args={{
    result,
    entityNoun: "bean",
    entityName: "Kiamugumo AB",
    backURL: "/beans/mika.arabica.example/bean-rkey",
  }}
/>
