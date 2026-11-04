<script module lang="ts">
  import { defineMeta } from "@storybook/addon-svelte-csf";
  import RecordViewHeader from "./RecordViewHeader.svelte";

  const { Story } = defineMeta({
    title: "Components/RecordViewHeader",
    component: RecordViewHeader,
    tags: ["autodocs"],
  });

  const iso = "2026-11-01T08:30:00Z";
</script>

<Story
  name="Community record"
  args={{
    recordType: "brew",
    title: "Espresso, Kiamugumo AB",
    timestamp: "November 1, 2026 at 8:30 AM",
    timestampISO: iso,
    authorDID: "did:plc:author1",
    authorHandle: "mika.arabica.example",
    authorDisplay: "Mika",
  }}
/>
<Story
  name="Own record"
  args={{
    recordType: "bean",
    title: "Kiamugumo AB",
    timestamp: "October 12, 2026 at 9:00 PM",
    timestampISO: "2026-10-12T21:00:00Z",
  }}
/>
